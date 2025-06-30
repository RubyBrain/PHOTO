package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/your-org/photo-booking-api-gateway/internal/config"
	"github.com/your-org/photo-booking-api-gateway/internal/controllers"
	"github.com/your-org/photo-booking-api-gateway/internal/middleware"
	"github.com/your-org/photo-booking-api-gateway/internal/services"
	"github.com/your-org/photo-booking-api-gateway/pkg/grpc_client"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.Load("./configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Инициализация логгера
	initLogger(cfg)

	// Создание gRPC соединений
	grpcConns, err := initGRPCConnections(cfg)
	if err != nil {
		log.Fatalf("Failed to establish gRPC connections: %v", err)
	}
	defer func() {
		for _, conn := range grpcConns {
			if err := conn.Close(); err != nil {
				log.Printf("Failed to close gRPC connection: %v", err)
			}
		}
	}()

	// Инициализация сервисов
	authService := services.NewAuthService(cfg.Auth.JWTSecret, cfg.Auth.TokenExpiry)
	bookingClient := grpc_client.NewBookingClient(grpcConns["booking"])
	scheduleClient := grpc_client.NewScheduleClient(grpcConns["schedule"])

	// Инициализация контроллеров
	bookingController := controllers.NewBookingController(bookingClient, scheduleClient)
	authController := controllers.NewAuthController(authService)

	// Настройка Gin
	router := setupRouter(cfg, authService, bookingController, authController)

	// Запуск сервера с graceful shutdown
	runServerWithGracefulShutdown(cfg, router)
}

func initLogger(cfg *config.Config) {
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting server in %s mode", cfg.Server.Env)
}

func initGRPCConnections(cfg *config.Config) (map[string]*grpc.ClientConn, error) {
	conns := make(map[string]*grpc.ClientConn)

	services := map[string]string{
		"booking":    cfg.Services.BookingService,
		"schedule":   cfg.Services.ScheduleService,
		"notification": cfg.Services.NotificationService,
	}

	for name, addr := range services {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(
			ctx,
			addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to %s service: %w", name, err)
		}

		conns[name] = conn
		log.Printf("Connected to %s service at %s", name, addr)
	}

	return conns, nil
}

func setupRouter(
	cfg *config.Config,
	authService *services.AuthService,
	bookingController *controllers.BookingController,
	authController *controllers.AuthController,
) *gin.Engine {
	router := gin.New()

	// Глобальные middleware
	router.Use(
		gin.Recovery(),
		middleware.RequestLogger(cfg.Logging.Format),
		middleware.ErrorHandler(),
		middleware.PrometheusMetrics(),
		cors.New(cors.Config{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			MaxAge:           12 * time.Hour,
		}),
	)

	// Health checks
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Metrics endpoint
	if cfg.Monitoring.EnableMetrics {
		router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// API routes
	api := router.Group("/api/v1")
	{
		// Auth routes
		auth := api.Group("/auth")
		{
			auth.POST("/login", authController.Login)
			auth.POST("/refresh", authController.RefreshToken)
		}

		// Authenticated routes
		authenticated := api.Group("")
		authenticated.Use(middleware.AuthMiddleware(authService))
		{
			// Booking routes
			bookings := authenticated.Group("/bookings")
			{
				bookings.POST("", bookingController.CreateBooking)
				bookings.GET("", bookingController.GetUserBookings)
				bookings.GET("/:id", bookingController.GetBookingDetails)
				bookings.DELETE("/:id", bookingController.CancelBooking)
			}

			// Profile routes
			profile := authenticated.Group("/profile")
			{
				profile.GET("", authController.GetProfile)
			}
		}

		// Admin routes
		admin := api.Group("/admin")
		admin.Use(middleware.AuthMiddleware(authService), middleware.AdminOnly())
		{
			admin.GET("/bookings", bookingController.ListAllBookings)
		}
	}

	return router
}

func runServerWithGracefulShutdown(cfg *config.Config, router *gin.Engine) {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.Timeout,
		WriteTimeout: cfg.Server.Timeout,
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		log.Printf("Server starting on port %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server failed: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		<-gCtx.Done()
		log.Println("Shutting down server...")
		
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := srv.Shutdown(ctx); err != nil {
			return fmt.Errorf("forced shutdown: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		log.Printf("Exit reason: %v\n", err)
	}
}
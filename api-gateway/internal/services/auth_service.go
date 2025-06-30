package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthService struct {
	jwtSecret      string
	accessExpiry   time.Duration
	refreshExpiry  time.Duration
	refreshTokens  map[string]time.Time
}

func NewAuthService(jwtSecret string, accessExpiry, refreshExpiry time.Duration) *AuthService {
	return &AuthService{
		jwtSecret:      jwtSecret,
		accessExpiry:   accessExpiry,
		refreshExpiry:  refreshExpiry,
		refreshTokens:  make(map[string]time.Time),
	}
}

type TokenPair struct {
	AccessToken   string `json:"access_token"`
	RefreshToken  string `json:"refresh_token"`
	AccessExpiry  int64  `json:"access_expiry"`
	RefreshExpiry int64  `json:"refresh_expiry"`
}

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func (s *AuthService) GenerateTokenPair(userID, role string) (*TokenPair, error) {
	accessToken, err := s.generateToken(userID, role, s.accessExpiry)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateToken(userID, role, s.refreshExpiry)
	if err != nil {
		return nil, err
	}

	// Сохраняем refresh token
	s.refreshTokens[refreshToken] = time.Now().Add(s.refreshExpiry)

	return &TokenPair{
		AccessToken:   accessToken,
		RefreshToken:  refreshToken,
		AccessExpiry:  time.Now().Add(s.accessExpiry).Unix(),
		RefreshExpiry: time.Now().Add(s.refreshExpiry).Unix(),
	}, nil
}

func (s *AuthService) generateToken(userID, role string, expiry time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "photo-booking-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

func (s *AuthService) RefreshToken(refreshToken string) (*TokenPair, error) {
	// Проверяем существование refresh token
	if _, ok := s.refreshTokens[refreshToken]; !ok {
		return nil, errors.New("invalid refresh token")
	}

	// Валидируем токен
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		delete(s.refreshTokens, refreshToken)
		return nil, err
	}

	// Удаляем использованный refresh token
	delete(s.refreshTokens, refreshToken)

	// Генерируем новую пару токенов
	return s.GenerateTokenPair(claims.UserID, claims.Role)
}
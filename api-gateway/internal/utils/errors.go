package utils

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code       int         `json:"code"`
	Message    string      `json:"message"`
	Details    interface{} `json:"details,omitempty"`
	Internal   error       `json:"-"`
	StackTrace string      `json:"-"`
}

func (e AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%d: %s (internal: %v)", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

func NewError(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

func WithDetails(err *AppError, details interface{}) *AppError {
	return &AppError{
		Code:    err.Code,
		Message: err.Message,
		Details: details,
	}
}

func WithInternal(err *AppError, internal error) *AppError {
	return &AppError{
		Code:     err.Code,
		Message:  err.Message,
		Internal: internal,
	}
}

var (
	ErrUnauthorized        = NewError(http.StatusUnauthorized, "Необходима авторизация")
	ErrInvalidToken        = NewError(http.StatusUnauthorized, "Неверный токен")
	ErrExpiredToken        = NewError(http.StatusUnauthorized, "Токен истек")
	ErrForbidden           = NewError(http.StatusForbidden, "Доступ запрещен")
	ErrNotFound            = NewError(http.StatusNotFound, "Ресурс не найден")
	ErrBadRequest          = NewError(http.StatusBadRequest, "Неверный запрос")
	ErrValidationFailed    = NewError(http.StatusUnprocessableEntity, "Ошибка валидации")
	ErrConflict            = NewError(http.StatusConflict, "Конфликт данных")
	ErrTooManyRequests     = NewError(http.StatusTooManyRequests, "Слишком много запросов")
	ErrInternalServerError = NewError(http.StatusInternalServerError, "Внутренняя ошибка сервера")
	ErrServiceUnavailable  = NewError(http.StatusServiceUnavailable, "Сервис недоступен")
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, ginErr := range c.Errors {
			switch err := ginErr.Err.(type) {
			case *AppError:
				c.JSON(err.Code, err)
			default:
				c.JSON(http.StatusInternalServerError, WithInternal(ErrInternalServerError, err))
			}
		}
	}
}
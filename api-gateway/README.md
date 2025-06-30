# API Gateway
Микросервис для маршрутизации запросов к основным сервисам.

## Технологии
- Go 1.21
- Gin
- gRPC (опционально)

## Запуск
```bash
go run main.go

## Архитектура
- Основное приложение (MVC): `models/`, `controllers/`  
- Микросервис API Gateway: [`api-gateway/`](/api-gateway/README.md)  

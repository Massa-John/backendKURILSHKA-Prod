# backendKURILSHKA-Prod

Production версия Go API для чат-приложения с поддержкой PostgreSQL и Redis.

## Описание

- **Язык:** Go 1.22
- **Фреймворк:** Gorilla Mux
- **БД:** PostgreSQL 16
- **Кэш:** Redis 7
- **Порт:** 8080 (по умолчанию)

## Эндпоинты

- `POST /login` — вход по номеру телефона и пароля
- `POST /api/change-password` — смена пароля (требует токен)
- `GET /api/users` — список всех пользователей
- `GET /api/contacts` — список контактов с последними сообщениями
- `GET /api/messages` — все сообщения
- `POST /api/messages` — отправить сообщение
- `GET /health` — статус сервиса

## Переменные окружения

```bash
PORT=8080
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=chatdb
POSTGRES_USER=chatuser
POSTGRES_PASSWORD=chatpass
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=redispass
```

## Локальный запуск

```bash
go mod download
go run main.go
```

## Docker

Этот сервис запускается через основной `docker-compose.yml` из корневой папки проекта.

# BAZA

Docker infrastructure for PostgreSQL and Redis.

## Run

```bash
docker network create app-network || true
cp .env.example .env
docker compose up -d
```

## PostgreSQL

- Host: postgres
- Port: 5432
- Database: chatdb
- User: chatuser
- Password: chatpass

## Redis

- Host: redis
- Port: 6379
- Password: redispass

## Useful commands

```bash
docker compose ps
docker compose logs -f postgres
docker compose logs -f redis
docker compose down -v
```

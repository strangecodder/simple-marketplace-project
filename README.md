# Marketplace Demo

Пример микросервисной реализации маркетплейса. Состоит из трёх доменных сервисов — **листинг**, **заказы** и **оплата** — плюс инфраструктура: PostgreSQL (по одной БД на сервис), RabbitMQ как брокер сообщений и уведомления через отдельный сервис-консьюмер.

## Архитектура

| Сервис | Порт (host) | Назначение | Зависимости |
|---|---|---|---|
| `listing-service` | 8080 | Каталог товаров/объявлений | `listing-db` |
| `order-service` | 8082 | Оформление и обработка заказов | `order-db`, `rabbitmq`, `listing-service`, `payment-service` |
| `payment-service` | 8083 | Обработка платежей | `payment-db` |
| `notification-service` | 8081 | Отправка уведомлений (consumer) | `rabbitmq` |
| `listing-db` | 5432 | PostgreSQL для листинга | — |
| `order-db` | 5434 | PostgreSQL для заказов | — |
| `payment-db` | 5436 | PostgreSQL для оплаты | — |
| `rabbitmq` | 5672 (AMQP), 15672 (UI) | Брокер сообщений между сервисами | — |

Сервисы взаимодействуют друг с другом по gRPC, а также через очереди RabbitMQ (например, `order-service` публикует события, которые забирает `notification-service`).

Все сервисы поднимаются в строгом порядке через `depends_on` с условием `service_healthy` — то есть `order-service` не запустится, пока не станут по-настоящему готовы (а не просто "запущены") `order-db`, `rabbitmq`, `listing-service` и `payment-service`.

## Требования

- Docker Engine ≥ 20.10
- Docker Compose ≥ v2 (`docker compose`, не `docker-compose`)
- Свободные порты на хосте: `5432`, `5434`, `5436`, `5672`, `15672`, `8080`–`8083`
- Go 1.25+ — только если планируете собирать сервисы локально без Docker

## Структура проекта

```
.
├── docker-compose.yml
├── sql/
│   ├── listing-db.sql
│   ├── order-db.sql
│   └── payment-db.sql
└── services/
    ├── listing-service/
    ├── order-service/
    ├── payment-service/
    └── notification-service/
```

SQL-файлы в `sql/` выполняются автоматически при первом старте соответствующего контейнера PostgreSQL (через `docker-entrypoint-initdb.d`) — они применяются **только на пустом volume**, повторный запуск на уже существующей БД их не подхватит.

## Переменные окружения

Каждый сервис читает конфигурацию из окружения (см. `config` в коде каждого сервиса). Перед первым запуском заполните блоки `environment` в `docker-compose.yml` для `listing-service`, `payment-service` и `notification-service` — в текущем виде они содержат заглушки. Пример необходимых переменных:

```env
# listing-service
DATABASE_HOST=listing-db
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_NAME=listingDB
SERVER_PORT=8080

# order-service
DATABASE_HOST=order-db
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_NAME=orderDB
SERVER_PORT=8080
RABBIT_HOST=rabbitmq
RABBIT_PORT=5672
RABBIT_USER=guest
RABBIT_PASSWORD=guest

# payment-service
DATABASE_HOST=payment-db
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_NAME=paymentDB
SERVER_PORT=8080

# notification-service
RABBIT_HOST=rabbitmq
RABBIT_PORT=5672
RABBIT_USER=guest
RABBIT_PASSWORD=guest
SERVER_PORT=8080
```

Точный список переменных зависит от структуры `config.Config` в каждом сервисе — сверьтесь с исходным кодом (`cmd/`, `internal/config`).

## Запуск

1. Склонируйте репозиторий и перейдите в его корень:
   ```bash
   git clone <repo-url>
   cd <repo-name>
   ```

2. Заполните переменные окружения в `docker-compose.yml` (см. раздел выше).

3. Соберите образы и поднимите все сервисы:
   ```bash
   docker compose up --build
   ```
   Для запуска в фоне:
   ```bash
   docker compose up --build -d
   ```

4. Проверьте состояние контейнеров и их healthcheck-статус:
   ```bash
   docker compose ps
   ```
   Все сервисы должны быть в статусе `healthy`. Первый запуск может занять больше времени — БД инициализируются, RabbitMQ поднимается, затем стартуют доменные сервисы в порядке зависимостей.

5. Управление RabbitMQ доступно по адресу [http://localhost:15672](http://localhost:15672) (логин/пароль: `guest`/`guest`).

## Проверка работоспособности

Посмотреть логи конкретного сервиса:
```bash
docker compose logs -f order-service
```

Проверить health вручную через `grpc_health_probe` внутри контейнера:
```bash
docker compose exec order-service /bin/grpc_health_probe -addr=:8080
```

Подключиться к БД любого сервиса напрямую:
```bash
docker compose exec listing-db psql -U postgres -d listingDB
```

## Остановка и очистка

Остановить сервисы, сохранив данные в volumes:
```bash
docker compose down
```

Остановить и полностью удалить данные (volumes с PostgreSQL и RabbitMQ):
```bash
docker compose down -v
```

## Известные ограничения демо-проекта

- Нет TLS/аутентификации между сервисами — все соединения внутри Docker-сети открытые.
- Учётные данные (`postgres/postgres`, `guest/guest`) захардкожены и подходят только для локальной разработки.
- Нет централизованного логирования и трейсинга между сервисами.
- SQL-миграции применяются только при первом старте контейнера БД (пустой volume); для изменения схемы на уже существующей БД нужен отдельный миграционный инструмент.
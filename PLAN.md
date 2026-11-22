# Foodly — план учебного проекта

Это источник правды. Агент не добавляет сервисы, RPC, таблицы, топики и библиотеки, которых здесь нет. Если запрос пользователя расходится с планом, агент говорит об этом и ждёт, пока план обновят. Менять план можно только вместе с пользователем.

Код сервисов пишет пользователь. Агент даёт фрагмент в чат, объясняет и ревьюит. В файлы с логикой агент не пишет, пока пользователь прямо не попросит поправить конкретный файл.

## Где мы сейчас

Сделано:

- Контракты: `identity`, `catalog`, `order`, `payment`, `events`. `buf generate` уже проходил.
- Миграции всех сервисов лежат в `migrations/`, в базы вручную их не накатывали. `identity` создаёт `identity_db` и накатывает свою миграцию при старте.
- `identity`: работает только `Register` (gRPC `:50051`, пользователь `ada@example.com` уже есть).

Дальше, и только это: `Login` и `ValidateToken` в `identity`. Kafka, catalog и order не начинать, пока логин не проверен через `grpcurl`.

## Зачем проект

Учебный стенд доставки еды. Не университетская работа. Цель — самому провести Go-сервис через границы: своя база, gRPC-команда с ответом, событие в Kafka, согласованность без распределённой транзакции.

Готовый v1: регистрация → меню → заказ → резерв остатков → оплата (успех и отказ) → статус заказа → письмо в notification из Kafka. Publisher можно убить и поднять, событие не теряется.

## Как устроена папка

`academic_project` — workspace, не Go-модуль. Корневого `go.mod`, общего `pkg/` и общего Makefile нет.

| Папка | Роль |
|---|---|
| `contracts/` | Единственный общий модуль: protobuf и сгенерированный Go. Импорт `github.com/foodly/contracts`, локально через `replace => ../contracts`. |
| `infra/` | Позже: compose со стендом. Сейчас пустой. Это не приложение. |
| `identity/` | Пользователи, пароль, JWT. База `identity_db`. gRPC `:50051`. |
| `catalog/` | Рестораны, меню, остатки, резервы. База `catalog_db`. gRPC `:50052`. |
| `order/` | Заказ и saga. База `order_db`. gRPC `:50053`. |
| `payment/` | Эмуляция списания и возврата. База `payment_db`. gRPC `:50054`. |
| `notification/` | Только consumer Kafka. База `notification_db`. gRPC нет. |
| `gateway/` | HTTP для браузера, порт `:8080`. Своей базы нет. Ходит в identity, catalog, order по gRPC. |

Внутри сервиса:

```
service/
  go.mod
  cmd/<name>/main.go
  internal/config
  internal/postgres
  internal/<область>     # сценарий и репозиторий
  internal/grpc          # package grpcsrv, если есть gRPC
  migrations/
  Dockerfile
  Makefile
  README.md
```

`main` только собирает зависимости и слушает порт. SQL не попадает в gRPC-обработчик. Protobuf не попадает в репозиторий.

## Локальная машина

Зафиксировано по факту, не как пожелание:

- ОС Windows, команды в чате — PowerShell. JSON для `grpcurl` в одинарных кавычках, без `\"`.
- Go 1.27. В `go.mod` строка `go 1.27`.
- Postgres — контейнер `foodly-postgres`, с хоста порт **5433**. Внутри контейнера порт 5432. Чужой Postgres на 5432 (базы `practice`, `wallets_test`) не трогать.
- Адрес базы, порт gRPC, секрет JWT и прочие настройки процесса читаются из окружения. В исходниках нет значений по умолчанию.
- Базу создаёт процесс сервиса (`EnsureDatabase`), таблицы создаёт `golang-migrate` при старте. Ручной прогон SQL в pgAdmin не делаем.
- Redpanda поднимаем на этапе Kafka, не раньше. Пока её нет, это нормально.

## Стек, который уже выбран

Другое не подставлять «потому что тоже подходит».

| Задача | Выбор |
|---|---|
| Язык | Go 1.27 |
| Контракты | protobuf, `buf` v2, плагины `protoc-gen-go` и `protoc-gen-go-grpc` |
| RPC | `google.golang.org/grpc` |
| HTTP-вход | `chi` только в `gateway` |
| Postgres | `jackc/pgx/v5`, пул `pgxpool` |
| Миграции | `golang-migrate` v4, SQL вшит через `embed` + `iofs` (на Windows не используем `file://`) |
| Пароли | `golang.org/x/crypto/bcrypt`, `DefaultCost` |
| Токен | `github.com/golang-jwt/jwt/v5`, HMAC **HS256**, срок 24 часа, в claims только `sub`, `iat`, `exp` |
| UUID | `github.com/google/uuid`. Ключи UUID, не `serial` |
| Деньги | `int64` в минимальных единицах (`price_minor`, `amount_minor`). Не `float` |
| Kafka-клиент | `segmentio/kafka-go` |
| Брокер | Redpanda в Docker, совместима с протоколом Kafka |
| Конфиг | переменные окружения, файл `.env` через `github.com/joho/godotenv` |
| Логи | стандартный `log`, пока не подключаем zap/slog-обёртки |

Не в v1: Kubernetes, service mesh, CQRS, общий SDK, очереди кроме Kafka, Redis, gRPC-gateway, курьеры, отзывы, список заказов, `CancelOrder` кнопкой.

## Домен и кто с кем говорит

Клиент знает только gateway. Между сервисами нет HTTP и нет общих таблиц.

```mermaid
flowchart LR
  Client --> Gateway
  Gateway -->|gRPC| identity
  Gateway -->|gRPC| catalog
  Gateway -->|gRPC| order
  order -->|Reserve Release| catalog
  order -->|Charge Refund| payment
  order -->|outbox| Kafka
  payment -->|outbox| Kafka
  catalog -->|outbox| Kafka
  Kafka --> notification
  Kafka --> order
```

**gRPC** — команда, нужен ответ в этом же вызове.

**Kafka** — факт, который уже случился. Ответа нет, получателей может быть несколько.

`notification` никого не вызывает. `order` не импортирует notification и не знает, что письма существуют. `catalog` не хранит `order_id`: резерв — его собственная сущность.

Ошибки бизнес-правил — коды gRPC (`InvalidArgument`, `AlreadyExists`, `Unauthenticated`, `FailedPrecondition`, `ResourceExhausted`), не поля `ok` / `error` в ответе. Исключение одно: `ValidateToken` отвечает успехом и `is_valid=false`, потому что вопрос метода — «годный ли токен».

`Idempotency-Key` лежит в теле сообщения. В gRPC нет привычных HTTP-заголовков, ключ должен быть виден в контракте.

Повторный `Reserve`, `Charge`, `Refund`, `CreateOrder` с тем же ключом возвращает тот же результат и не списывает ресурс второй раз. Уникальный индекс в своей базе — настоящая защита. Проверка `SELECT` до `INSERT` не заменяет индекс.

## Контракты

Файлы уже написаны. Новые RPC и поля не добавлять без правки этого плана и `.proto`.

- `IdentityService`: `Register`, `Login`, `ValidateToken`.
- `CatalogService`: `ListRestaurants`, `GetMenu`, `ReserveItems`, `ReleaseItems`.
- `OrderService`: `CreateOrder`, `GetOrder`. Статусы `UNSPECIFIED`, `PENDING`, `CONFIRMED`, `CANCELLED`.
- `PaymentService`: `Charge`, `Refund`.
- `events.v1`: только `message`, без `service`. `EventMeta` (`event_id`, `occurred_at_unix_ms`) и факты `OrderCreated`, `OrderPaid`, `OrderCancelled`, `PaymentSucceeded`, `PaymentFailed`, `StockReserved`, `StockReleased`.

Цена в `CreateOrderRequest.items` клиентом не задаётся. Order сам читает цену и сохраняет снимок. Поле в proto общее для запроса и ответа; в сценарии создания цену из запроса игнорируют.

`user_id` в `CreateOrder` ставит gateway после `ValidateToken`. Клиентский JSON это поле не диктует.

## Данные

Отдельные базы: `identity_db`, `catalog_db`, `order_db`, `payment_db`, `notification_db`. Внешних ключей между базами нет. `user_id` в заказе — просто UUID.

Схемы уже лежат в миграциях. Кратко, чтобы не выдумать другие таблицы:

- identity: `users` (id, email, password_hash, name, created_at). Уникальность `lower(email)`.
- catalog: `restaurants`, `menu_items` (`price_minor`, `available_qty`), `reservations` (ключ идемпотентности, `released_at`), `reservation_items`, `outbox`. Сид: Pizza Place, Margherita 45000, Cola 15000.
- order: enum `order_status`, `orders`, `order_items` (снимок `price_minor`), `outbox`, `processed_events`.
- payment: enum `payment_status` (`succeeded`, `failed`, `refunded`), `payments`, `refunds`, `outbox`. Неуспешное списание тоже хранится, чтобы тот же ключ не списал деньги со второй попытки.
- notification: `processed_events`, `notifications`. Outbox нет: сервис ничего не публикует.

`outbox`: `id`, `event_type`, `payload bytea`, `created_at`, `published_at`. Неотправленные выбираются частичным индексом `published_at IS NULL`.

`down` миграции снимает объекты в обратном порядке. `DROP TABLE` для таблицы из `up` — правильный откат. Enum-тип снимается отдельным `DROP TYPE` после таблиц. Сид откатывается `DELETE` конкретных UUID, не `TRUNCATE`.

## Согласованность

Внутри сервиса — транзакция Postgres. Между сервисами двухфазной фиксации нет.

Saga создания заказа, один синхронный RPC `CreateOrder`:

1. Вставить заказ `pending` и позиции со снимком цены.
2. `catalog.ReserveItems`. Нет остатка — `cancelled`, клиенту ошибка.
3. `payment.Charge` на `total_price_minor`.
4. Успех — `confirmed`.
5. Отказ оплаты — `catalog.ReleaseItems`, статус `cancelled`.

До появления payment (этап order без Kafka) после успешного резерва заказ остаётся `pending`. В `confirmed` его не ставить: это сломает смысл статуса.

Событие пишется в `outbox` в той же транзакции, что и бизнес-строка. Отдельный цикл в том же процессе читает неотправленные строки и кладёт их в Kafka, затем ставит `published_at`.

Топики, по одному типу сообщения:

| Топик | Сообщение | Кто пишет | Кто читает |
|---|---|---|---|
| `foodly.order.created.v1` | OrderCreated | order | notification |
| `foodly.order.paid.v1` | OrderPaid | order | notification |
| `foodly.order.cancelled.v1` | OrderCancelled | order | notification |
| `foodly.payment.succeeded.v1` | PaymentSucceeded | payment | order, notification |
| `foodly.payment.failed.v1` | PaymentFailed | payment | order, notification |
| `foodly.catalog.stock-reserved.v1` | StockReserved | catalog | notification |
| `foodly.catalog.stock-released.v1` | StockReleased | catalog | notification |

Группа `notification-v1` читает все топики. Группа `order-payments-v1` читает только топики оплаты. Повтор доставки отсекается `processed_events.event_id`. После исчерпания повторов сообщение уходит в `<топик>.dlt`.

`StockReserved` не содержит `order_id`: catalog его не знает.

## HTTP gateway

Появится после того, как gRPC сервисов уже вызываются через `grpcurl`.

| Метод | Путь | Куда |
|---|---|---|
| POST | `/register` | identity.Register |
| POST | `/login` | identity.Login |
| GET | `/restaurants` | catalog.ListRestaurants |
| GET | `/restaurants/{id}/menu` | catalog.GetMenu |
| POST | `/orders` | order.CreateOrder |
| GET | `/orders/{id}` | order.GetOrder |

`POST /orders` требует `Authorization: Bearer`. Gateway вызывает `ValidateToken` и подставляет `user_id`. Невалидный токен — HTTP 401, до order запрос не доходит.

## Этапы

Каждый этап заканчивается проверкой из таблицы. Следующий не начинать раньше.

### 1. Identity

Сейчас. `Register` готов.

Осталось: `FindByEmail`, `Login`, JWT, `ValidateToken`. Секрет только из `JWT_SECRET`, не короче 32 байт. Пустой секрет — ошибка старта, не запасное значение в коде. Неверный email и неверный пароль снаружи одинаковы (`Unauthenticated`). Если пользователя нет, всё равно выполнить сравнение bcrypt с заранее посчитанным хешем, чтобы ответ не был мгновенным. Алгоритм токена проверять: принимать только HS256.

Проверка: `grpcurl` Login для `ada@example.com` / `secret12` возвращает токен; ValidateToken даёт `isValid: true` и тот же user id; чужой пароль — `Unauthenticated`; порванный токен — успех RPC и `isValid: false`.

### 2. Catalog

Чтение меню и резерв. Без Kafka: outbox-таблица есть, publisher не писать.

Сценарии: список ресторанов и меню отдают сид; `ReserveItems` уменьшает `available_qty` и возвращает `reservation_id`; повтор с тем же ключом не уменьшает остаток второй раз; нехватка — `FailedPrecondition` или `ResourceExhausted`; `ReleaseItems` возвращает количество и ставит `released_at`.

### 3. Order без оплаты и без Kafka

`CreateOrder` синхронно вызывает catalog. Статус после резерва — `pending`. Order не читает таблицы catalog. Цена считается на стороне order из данных каталога, не из запроса клиента. Свой `idempotency_key` заказа и отдельный ключ резерва.

Проверка: два одинаковых создания заказа дают один `order_id`; остаток уменьшается один раз; `GetOrder` показывает снимок цены.

### 4. Payment и saga

`Charge` / `Refund`. Отказ оплаты компенсируется `ReleaseItems` и статусом `cancelled`. Успех — `confirmed`. Ключи идемпотентности разные у списания и возврата.

Проверка: сценарий отказа возвращает остаток; повторный Charge с тем же ключом не создаёт второе списание.

### 5. Kafka, outbox, notification

Поднять Redpanda. Publisher в order, payment, catalog. Notification пишет строку письма и `processed_events`. Order догоняет статус по событиям оплаты, если saga ещё не обновила его сама. Убить notification: checkout всё равно завершается. Поднять: письма догоняются. Убить процесс между commit и публикацией: после старта событие уходит в топик.

### 6. Устойчивость

Таймауты и ограниченные повторы gRPC. Повторная доставка Kafka. Грязная миграция не лечится удалением базы, пока не разобрана причина. Рестарт посередине saga не оставляет резерв без заказа и не списывает деньги дважды.

### 7. По желанию, не часть v1

Трейсы OpenTelemetry. Отдельный процесс publisher. Сервис курьера новым проектом в этой же папке. Свой git-репозиторий на сервис.

## Конфигурация

У каждого сервиса свой `.env` в корне этого сервиса. Процесс запускают из этой папки, чтобы `godotenv` нашёл файл. Уже заданные в системе переменные файл не перетирает.

В git попадает только `.env.example`: имена переменных и локальные учебные значения. `.env` в git не кладётся. В Go нет констант с URL, паролем, портом или секретом и нет подстановки «если пусто — вот это».

Нет переменной — процесс завершается с текстом, какой переменной не хватает.

Первый набор у identity:

| Переменная | Смысл |
|---|---|
| `DATABASE_URL` | Postgres, включая имя базы `identity_db` |
| `GRPC_ADDR` | адрес прослушивания, например `:50051` |
| `JWT_SECRET` | секрет HS256, не короче 32 байт |

Следующие сервисы заводят свой `.env` с теми же именами и своим портом и базой. Имена не выдумывать заново.

## Чего агент не делает по своей инициативе

- Не подставляет в код URL базы, адрес gRPC, пароль или секрет «на время учёбы».
- Не переписывает UUID на `int` / `bigint`.
- Не выносит общий логгер, outbox или interceptors в корневой `pkg`.
- Не склеивает сервисы в один `go.mod`.
- Не ставит `confirmed` сразу после резерва.
- Не добавляет CRUD ресторана: меню для учёбы берётся из сида.
- Не логирует пароль, токен и полный DSN.
- Не считает молчание пользователя согласием на новый этап.
- В PowerShell не экранирует кавычки обратным слэшем.

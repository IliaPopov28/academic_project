# Foodly — план учебного проекта

Это источник правды. Агент не добавляет сервисы, RPC, таблицы, топики и библиотеки, которых здесь нет. Если запрос пользователя расходится с планом, агент говорит об этом и ждёт, пока план обновят. Менять план можно только вместе с пользователем.

Код сервисов пишет пользователь. Агент даёт фрагмент в чат, объясняет и ревьюит. В файлы с логикой агент не пишет, пока пользователь прямо не попросит поправить конкретный файл.

## Где мы сейчас

**Этап 1 (identity) — закрыт.** Проверено через `grpcurl` (2026-11-25):

- Login `ada@example.com` / `secret12` → `accessToken`
- ValidateToken этим токеном → `isValid: true`, `userId` = `4de2ce99-efe1-47d3-85a7-db7116b0f777`
- неверный пароль → `Unauthenticated`
- битый токен → успешный RPC, в JSON часто `{}` (поля с нулями не печатаются; это `is_valid=false`)

Также сделано раньше:

- Контракты всех сервисов + `buf generate`
- SQL-миграции identity / catalog / order / payment / notification в файлах
- Репозиторий на GitHub: https://github.com/IliaPopov28/academic_project (публичный). `.env` и `*.exe` в git не попадают
- Postgres Foodly: контейнер `foodly-postgres`, порт **5433**. Перед стартом сервисов: `docker start foodly-postgres`

**Дальше — только этап 2: catalog.** Не начинать order, payment, Kafka, gateway, Dockerfile/Makefile сервисов.

Порядок работ по catalog (не перескакивать):

1. Каркас как у identity: `go.mod`, `.env` / `.env.example`, `config`, `postgres` (EnsureDatabase + Migrate + NewPool), `migrations/fs.go`, `main` без gRPC → проверка: база `catalog_db`, сид Pizza Place / Margherita / Cola
2. Чтение: репозиторий + сценарий + gRPC `ListRestaurants`, `GetMenu` → проверка `grpcurl`
3. Запись: `ReserveItems` / `ReleaseItems` с идемпотентностью и транзакцией → проверка `grpcurl`
4. Outbox-таблица уже в миграции; **publisher Kafka не писать** до этапа 5

`Dockerfile` и `Makefile` у сервисов — пустые заготовки. Наполнять после этапа Kafka / когда соберём compose в `infra`.

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

### 1. Identity — готово

Что лежит в коде:

- `Register`, `Login`, `ValidateToken` (gRPC + сценарий + репозиторий)
- JWT HS256, секрет из `JWT_SECRET` (≥ 32 байт), срок 24 часа, claims: `sub` / `iat` / `exp`
- `dummyHash` считается при старте из `crypto/rand` (не строка из интернета)
- конфиг только из `.env` через `godotenv`, без дефолтов в Go
- миграции через `embed` + `iofs`

Замечания на ревью (не блокируют этап, поправить при случае):

- при неизвестном email в `Login` желательно всё же вызывать `bcrypt.CompareHashAndPassword(s.dummyHash, ...)` до `ErrUnauthenticated`
- в `FindByEmail` сравнивать с `pgx.ErrNoRows`, не `sql.ErrNoRows`

Проверка пройдена (см. «Где мы сейчас»).

### 2. Catalog — текущий этап

Без Kafka: outbox-таблица есть, publisher не писать. CRUD ресторанов не делать — только сид.

**Сид (фиксированные UUID из миграции `000002_seed`):**

| Что | UUID | Прочее |
|---|---|---|
| Pizza Place | `a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11` | |
| Margherita | `b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11` | `price_minor=45000`, qty 20 |
| Cola | `c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11` | `price_minor=15000`, qty 50 |

**Структура пакетов (как identity):**

```
catalog/
  go.mod
  .env.example          # DATABASE_URL, GRPC_ADDR=:50052
  cmd/catalog/main.go
  internal/config
  internal/postgres     # копия паттерна identity, импорт catalog/migrations
  internal/catalog      # или internal/menu — сценарий + репозиторий
  internal/grpc         # package grpcsrv
  migrations/           # 000001_init + 000002_seed уже есть
```

**Зависимости в go.mod:** `pgx/v5`, `golang-migrate`, `godotenv`, `google.golang.org/grpc`, `github.com/foodly/contracts` + `replace => ../contracts`, `google/uuid`.

**Поведение RPC:**

| RPC | Успех | Ошибки |
|---|---|---|
| `ListRestaurants` | список из сида | — |
| `GetMenu` | позиции ресторана | неизвестный ресторан → `NotFound` |
| `ReserveItems` | `reservation_id`, `available_qty` уменьшен | пустой ключ/позиции → `InvalidArgument`; нехватка → `FailedPrecondition` или `ResourceExhausted`; повтор того же `idempotency_key` → тот же `reservation_id`, остаток не трогать |
| `ReleaseItems` | пустой ответ, qty возвращён, `released_at` | нет резерва → `NotFound`; уже отпущен → идемпотентный успех по `release_idempotency_key` или повтор без двойного возврата |

`Reserve` и `Release` — в одной транзакции: строки меню с блокировкой (`FOR UPDATE`), запись `reservations` / `reservation_items`, изменение qty. События в outbox на этом этапе можно не писать (таблица пустая), пока не дойдём до Kafka.

**Проверка каркаса:**

```powershell
docker start foodly-postgres
cd catalog
Copy-Item .env.example .env   # если ещё нет
go run ./cmd/catalog
```

В pgAdmin на `5433`: база `catalog_db`, две строки в `menu_items`.

**Проверка чтения (после gRPC):**

```powershell
grpcurl -plaintext -import-path .\contracts\proto -proto catalog/v1/catalog.proto -d '{}' localhost:50052 catalog.v1.CatalogService/ListRestaurants

grpcurl -plaintext -import-path .\contracts\proto -proto catalog/v1/catalog.proto -d '{"restaurant_id":"a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"}' localhost:50052 catalog.v1.CatalogService/GetMenu
```

**Проверка резерва (после Reserve/Release):** один `Reserve` уменьшает qty; повтор с тем же `idempotency_key` — тот же `reservation_id`, qty не меняется; `Release` возвращает qty; повторный `Release` не накручивает остаток.

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

Переменные по сервисам (имена не выдумывать заново):

| Сервис | Переменные |
|---|---|
| identity | `DATABASE_URL` → `identity_db`, `GRPC_ADDR=:50051`, `JWT_SECRET` (≥ 32 байт) |
| catalog | `DATABASE_URL` → `catalog_db`, `GRPC_ADDR=:50052` (JWT нет) |
| order | позже: `DATABASE_URL` → `order_db`, `GRPC_ADDR=:50053`, адреса catalog/payment |
| payment | позже: `DATABASE_URL` → `payment_db`, `GRPC_ADDR=:50054` |
| notification | позже: `DATABASE_URL` → `notification_db`, брокеры Kafka |
| gateway | позже: `HTTP_ADDR=:8080`, адреса identity/catalog/order |

Локальный учебный DSN (только в `.env` / `.env.example`, не в Go):

`postgres://postgres:postgres@localhost:5433/<имя_базы>?sslmode=disable`

## Документация и права агента на `.md`

Пользователь разрешил править в этом проекте markdown без отдельной просьбы каждый раз:

- `PLAN.md` — после подтверждённой проверки этапа обновлять «Где мы сейчас»; детали этапа расписывать заранее, чтобы не забывать
- `README.md` — краткая навигация, без дублирования всего плана
- `.cursor/rules/*.mdc` — правила обучения и следования плану

Код Go / SQL / proto по-прежнему пишет пользователь из фрагментов в чате, пока не попросит правку конкретного файла.

## Чего агент не делает по своей инициативе

- Не подставляет в код URL базы, адрес gRPC, пароль или секрет «на время учёбы».
- Не переписывает UUID на `int` / `bigint`.
- Не выносит общий логгер, outbox или interceptors в корневой `pkg`.
- Не склеивает сервисы в один `go.mod`.
- Не ставит `confirmed` сразу после резерва.
- Не добавляет CRUD ресторана: меню для учёбы берётся из сида.
- Не логирует пароль, токен и полный DSN.
- Не считает молчание пользователя согласием на новый этап.
- Не наполняет `Dockerfile` / `Makefile` сервисов до этапа infra / контейнеризации.
- В PowerShell не экранирует кавычки обратным слэшем.

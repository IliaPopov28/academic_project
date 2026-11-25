# Foodly — учебный workspace

**Это не monorepo.** `academic_project` — папка отдельных проектов. У каждого сервиса свой `go.mod`, миграции и Dockerfile. Корневого `go.mod` и общего `pkg/` нет.

Источник правды — [PLAN.md](PLAN.md): что сделано, какой шаг следующий, проверки `grpcurl`, UUID сида. Новые сервисы, RPC и библиотеки сначала в план, потом в код.

Сейчас: **identity готов** (Register / Login / ValidateToken). Дальше — **catalog** (меню и резерв, без Kafka). GitHub: https://github.com/IliaPopov28/academic_project

## Папки

| Папка | Что это |
|---|---|
| `contracts/` | Protobuf и сгенерированный Go. Единственный общий модуль. |
| `infra/` | Позже локальный стенд. Не приложение. |
| `identity/` | Регистрация, логин, JWT. Порт `:50051`, база `identity_db`. |
| `catalog/` | Меню и остатки. `:50052`, `catalog_db`. |
| `order/` | Заказ и saga. `:50053`, `order_db`. |
| `payment/` | Списание. `:50054`, `payment_db`. |
| `notification/` | Только Kafka. `notification_db`. |
| `gateway/` | HTTP `:8080` для клиента. Своей базы нет. |

## Как пишется код

Рабочий код переписывает пользователь из фрагментов в чате. Агент правит файлы сервисов только по явной просьбе. Postgres для Foodly — контейнер на порту **5433**.

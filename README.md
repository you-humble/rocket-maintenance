# Rocket Maintenance

![Coverage](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/you-humble/e0883b0b3af0adeffb3b01446a0d4223/raw/coverage.json)


```markdown
# Rocket Maintenance — monorepo микросервисов

Monorepo на Go с несколькими сервисами и общими библиотеками. Репозиторий содержит:
- **микросервисы**: `core`, `assembly`, `inventory`, `order`, `payment`, `iam`, `notification`
- **модули**: `assembly inventory order payment notification shared platform`
- **генерацию API/Proto**, **линт/форматирование**, **юнит/интеграционные тесты**, **docker-compose окружение**, **миграции**.

Управление типовыми операциями выполняется через **Taskfile**.

---

## Требования

- Go (используется версия из Taskfile: `GO_VERSION=1.25.4`)
- Docker + Docker Compose
- Node.js (нужен для `redocly` при бандле OpenAPI)
- Утилиты (устанавливаются тасками в локальный `./bin`): `golangci-lint`, `gofumpt`, `gci`, `buf`, `protoc` плагины, `grpcurl`, `ogen`, `yq`, `mockery`, `goose`, `envsubst`

---

## Быстрый старт

### 1) Поднять окружение (все сервисы)
```bash
task up-all
```

Остановить и удалить (включая volumes):
```bash
task down-all
```

### 2) Запуск отдельных сервисов локально (без Docker)
Inventory:
```bash
task run-inventory
```

Order:
```bash
task run-order
```

Payment:
```bash
task run-payment
```

---

## Структура репозитория (в общих чертах)

- `assembly/`, `inventory/`, `order/`, `payment/`, `notification/` — сервисы/модули
- `shared/` — общие зависимости (в т.ч. `shared/proto`)
- `platform/` — инфраструктурные/общие пакеты
- `deploy/compose/*` — compose-окружения по сервисам
- `deploy/env` — генерация `.env` файлов из шаблонов

---

## Основные команды (Taskfile)

### Форматирование кода
Форматирует все Go-файлы во всех модулях (кроме `*/mocks/*`) через `gofumpt` + сортировка импортов через `gci`:
```bash
task format
```

### Линт
Запускает `golangci-lint` по всем модулям:
```bash
task lint
```

### Генерация моков
Генерация моков интерфейсов через `mockery`:
```bash
task mockery:gen
```

### Генерация OpenAPI → Go (ogen)
1) Бандлит OpenAPI декларации (через `redocly`)
2) Находит YAML с `x-ogen:` и генерирует Go-код в `target/package`, указанные в файлах
```bash
task ogen:gen
```

### Генерация Proto → Go (buf)
Генерация Go-кода из `.proto` (директория `shared/proto`):
```bash
task proto:gen
```

### Полная генерация (Proto + OpenAPI)
```bash
task gen
```

---

## Тесты

### Юнит-тесты (без integration)
Запускает тесты по всем модулям и исключает каталоги интеграционных тестов:
```bash
task test
```

### Интеграционные тесты (по build tag `integration`)
Ищет файлы с `//go:build integration` и запускает тесты из `./<module>/tests/...`:
```bash
task test-integration
```

> При необходимости можно ограничить набор модулей, переопределив переменную `MODULES`:
```bash
task test MODULES="order payment"
```

---

## Покрытие тестами

### Сводное покрытие (и по каждому модулю)
Считает покрытие только для бизнес-пакетов (`internal/service`), исключая `mocks/testdata/pkg/transport/repository/proto/pb/cmd` и др.  
Файлы покрытий складываются в `./coverage`, затем склеиваются в `coverage/total.out`.
```bash
task test-coverage
```

### HTML-отчёт покрытия
Генерирует HTML и пытается открыть в браузере:
```bash
task coverage:html
```

---

## Docker Compose окружение

Compose окружения разнесены по директориям:
- `deploy/compose/core` — базовые зависимости (например, БД/брокеры и т.п.)
- `deploy/compose/<service>` — сервис + зависимости

Команды:

Core:
```bash
task up-core
task down-core
```

Сервисы (пример для order):
```bash
task up-order
task down-order
```

Все сервисы:
```bash
task up-all
task down-all
```

---

## Миграции (Order)

Используется `goose`. Директория миграций:
- `ORDER_MIGRATIONS_DIR=./order/migrations`

DSN по умолчанию:
- `POSTGRES_DSN=postgres://order-service-user:order-service-password@localhost:5432/order-service?sslmode=disable`

Статус:
```bash
task migrations-status
```

Применить:
```bash
task migrations-up
```

Откатить одну:
```bash
task migrations-down
```

Полный reset:
```bash
task migrations-reset
```

Создать новую SQL-миграцию:
```bash
task migrations-create -- add_users_table
```

---

## Генерация .env файлов

Генерирует `.env` для сервисов из `deploy/env/.env` и `deploy/env/.env.template` (если `.env` отсутствует — создаётся из template).  
Также прокидывает переменную `SERVICES` (список сервисов) в скрипт генерации.
```bash
task env:generate
```

---

## API smoke-тест (gRPC + REST)

Таск `test-api` прогоняет сценарий:
1) gRPC: Inventory `ListParts`, затем `GetPart`
2) REST: Order — создать заказ, проверить статус, оплатить, дождаться `COMPLETED` (polling), создать второй заказ и отменить

Запуск:
```bash
task test-api
```

Ожидаемые зависимости по адресам (как минимум из сценария):
- Inventory gRPC: `localhost:50051`
- Order REST: `http://localhost:8080`

---

## Полезные заметки

- Инструменты ставятся локально в `./bin` (переменная `BIN_DIR`)
- В большинстве тасков по модулям используется список `MODULES`:
  `assembly inventory order payment notification shared platform`
- Для форматирования импортов используется префикс:
  `github.com/you-humble/rocket-maintenance`

---

## Типовой рабочий цикл

1) Поднять окружение:
```bash
task up-all
```

2) Сгенерировать код (proto + openapi):
```bash
task gen
```

3) Привести код в порядок:
```bash
task format
task lint
```

4) Прогнать тесты:
```bash
task test
task test-integration
task test-coverage
```

5) Быстрая проверка API:
```bash
task test-api
```

---

## Troubleshooting

- Если `test-api` не дождался `COMPLETED`, проверь:
  - запущены ли consumers у `order` и `assembly`
  - корректны ли топики Kafka и `group.id`
  - сервисы доступны на ожидаемых портах
- Если генерация OpenAPI не происходит — убедись, что в YAML есть секция `x-ogen:` и корректно указаны `target` и `package`.

---
```
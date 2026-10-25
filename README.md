# Task Manager

[![ci](https://github.com/crypolt/taskmanager/actions/workflows/ci.yml/badge.svg)](https://github.com/crypolt/taskmanager/actions/workflows/ci.yml)

Локальный агентный таск-менеджер: задача → план → исполнение → проверка, с эскалацией
моделей по цепочке и канбан-доской в браузере. Бэкенд на Go (fiber + fx), UI на Vue 3,
исполнитель — CLI [opencode](https://opencode.ai), состояние — SQLite.

> An agentic task runner: a task is planned, executed by a coding agent, verified by a
> second model, and escalated down a per-role model chain. Go backend, Vue 3 board, SQLite.

## Возможности

- **Конвеер по стадиям**: planner → executor → verifier, при провале — следующая модель в цепочке роли.
- **Канбан на 7 колонок** с drag&drop, прогрессом, ETA и живыми попытками (SSE).
- **Дерево задач**: подзадачи из плана, агрегация результата, resume после паузы.
- **Учёт токенов и стоимости** по моделям, включая кеш (`cache.read`/`cache.write`).
- **Управление моделями** во вкладке «Модели»: цепочки ролей, порядок, цены, дневной лимит токенов, статус отказов провайдера.
- **Отмена убивает процесс** агента, а не только помечает задачу.
- **Fast-fail на отказ провайдера** (квота, rate limit, ключ) — задача падает за секунды, а не по таймауту.
- **Локализация** RU/EN: отдельные каталоги для бэкенда и фронтенда, гейт `check:locales`.

## Требования

- Go 1.27+
- Node.js 22+ и npm
- CLI `opencode` в `PATH` (или абсолютный путь в `OPENCODE_BIN`)

## Быстрый старт

```bash
git clone <repo-url> task-manager && cd task-manager
make install                 # go mod download + npm install
cp .env.example .env         # поправьте под себя
make run                     # соберёт UI при необходимости и поднимет :8080
```

Откройте <http://localhost:8080> — это доска. API живёт под `/api/v1`.

Разработка с горячей перезагрузкой UI:

```bash
make dev    # API на :8080 + Vite на :5173
```

Сборка бинарника с встроенным UI:

```bash
make build  # bin/taskmanager, фронтенд упакован через go:embed
```

## Конфигурация

Все переменные читаются через `kenv` из окружения или `.env` (см. `.env.example`).
Обязательных секретов нет: по умолчанию используется `opencode` со своим логином.

| Переменная | По умолчанию | Назначение |
| --- | --- | --- |
| `SERVICE_NAME` | `task-manager` | имя сервиса в логах |
| `LANGUAGE` | `ru` | язык сообщений бэкенда (`ru`/`en`) |
| `SERVER_ADDRESS` | `0.0.0.0:8080` | адрес HTTP-сервера |
| `LOG_LEVEL` | `debug` | `debug`/`info`/`warn`/`error` |
| `TRACE_MODE` | `true` | подробные трейсы запросов |
| `DB_PATH` | `taskmanager.db` | файл SQLite |
| `WORKERS` | `3` | параллельных задач |
| `MAX_PROMPT_CHARS` | `40000` | обрезка промпта по символам |
| `MAX_PROMPT_TOKENS` | `8000` | обрезка промпта по токенам |
| `EXEC_TIMEOUT_SEC` | `600` | жёсткий таймаут одного запуска агента |
| `OPENCODE_BIN` | `opencode` | путь к CLI |
| `OPENCODE_AUTO_APPROVE` | `true` | `--auto` (иначе headless упадёт на permission-ask) |
| `OPENCODE_PURE` | `true` | `--pure` |
| `OPENCODE_MCP` | `false` | включать MCP-серверы в генерируемом конфиге |
| `OPENCODE_STEPS` | `40` | лимит шагов агента (12 не хватает: агент умирает на записи файлов) |
| `OPENCODE_IDLE_SEC` | `120` | watchdog: нет событий → процесс убивается |
| `OPENCODE_CONFIG_PATH` | — | свой конфиг вместо сгенерированного |
| `OPENAI_BASE_URL` / `OPENAI_API_KEY` | — | провайдер `openai` для маршрутов |
| `OLLAMA_BASE_URL` | `http://localhost:11434` | провайдер `ollama` |

## API

Все ответы — JSON, ошибки вида `{"code","key","message"}`.

| Метод | Путь | Назначение |
| --- | --- | --- |
| GET | `/api/v1/health` | health-check |
| GET | `/api/v1/stats` | сводка по задачам и стоимости |
| GET | `/api/v1/pipelines` | доступные конвееры |
| GET/POST | `/api/v1/tasks` | список / создание задачи |
| GET | `/api/v1/tasks/:id` | задача с подзадачами и попытками |
| POST | `/api/v1/tasks/:id/cancel` | отменить (убивает процесс агента) |
| POST | `/api/v1/tasks/:id/retry` | перезапустить |
| POST | `/api/v1/tasks/:id/answer` | ответить на вопрос агента |
| POST | `/api/v1/tasks/:id/move` | перенос по колонкам |
| POST | `/api/v1/plans/preview` | предпросмотр плана |
| POST | `/api/v1/plans` | создать план подзадач |
| GET | `/api/v1/models` | цепочки моделей по ролям |
| POST | `/api/v1/models` | создать/заменить звено (`role`, `position`, `provider`, `model`, `price_in`, `price_out`, `daily_tokens`) |
| POST | `/api/v1/models/entries` | удалить звено (`role`, `position`), позиции схлопываются |
| POST | `/api/v1/models/move` | поменять местами (`role`, `from`, `to`) |
| GET | `/api/v1/models/usage` | расход по моделям за всё время |
| GET | `/api/v1/models/usage/daily` | расход за 24 часа |
| GET | `/api/v1/models/health` | последний отказ провайдера на `provider:model` |
| GET | `/api/v1/mcp` | список MCP-серверов |
| GET | `/api/v1/events` | SSE: `task`, `attempt`, `event` |

## Архитектура

```
main.go                     только DI-провайдеры (fx)
config/                     kenv-конфиг и валидация
models/                     доменные структуры, роли, пайплайны, read-модели
server/                     fiber: handlers, middlewares, routes
internal/services/
  internals/                engine, scheduler, planner, executor, verifier, state
  externals/llm/            провайдеры: opencode, openai, ollama, mock
internal/repository/        SQLite: задачи, попытки, маршруты, статистика
skills/agents/              промпты ролей (go:embed)
utils/                      i18n, логгер, токенизация, helpers
web/                        Vue 3: domain / application / infrastructure / presentation
assets/dist/                собранный UI (go:embed)
```

Слои: `presentation → application → domain`, инфраструктура подключается через DI.
Модели и константы — в `models/`, никаких «магических» строк в слоях.

## Модели и квоты

Каждая роль имеет упорядоченную цепочку `provider/model`. Попытка N берёт звено N,
при провале — следующее. Во вкладке «Модели» можно добавлять/удалять/менять порядок
звеньев, задавать цены и **дневной лимит токенов** (`daily_tokens`).

Лимит сейчас **отображается** (прогресс-бар «за сутки: X из Y»), но не блокирует
выбор модели: провайдерские квоты через CLI не читаются, а свой счётчик можно
использовать для ручного переключения.

## Тесты

```bash
make test    # go test ./... + npm run check:locales
make vet
cd web && npm test   # node --test для фронтенд-утилит
```

## Ограничения

- Один пользователь, без авторизации: сервис рассчитан на локальный запуск.
- SQLite без миграционного фреймворка: схема создаётся и добивается при старте.
- Без Docker/CI-деплоя: только локальная сборка.

## Ветки и участие

`main` — стабильные релизы, `develop` — интеграция, работа идёт в `feature/*` и `fix/*`
и попадает в `develop` через PR; релизы — `release/*`, срочные правки — `hotfix/*`.
Полные правила, конвенция коммитов и чек-лист перед PR — в [CONTRIBUTING.md](CONTRIBUTING.md).

## Лицензия

MIT — см. [LICENSE](LICENSE).

SHELL := /bin/bash

APP := taskmanager
BIN := bin/$(APP)
WEB := web
DIST := assets/dist

.DEFAULT_GOAL := help

.PHONY: help install web build run dev test vet fmt tidy clean db-reset

help: ## Показать доступные команды
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

install: ## Установить зависимости Go и веба
	go mod download
	cd $(WEB) && npm install

web: ## Собрать Vue-интерфейс в assets/dist
	cd $(WEB) && npm run build

build: web ## Собрать бинарник (UI встроен в него)
	go build -o $(BIN) .

run: ## Запустить проект (при отсутствии UI соберёт его)
	@test -f $(DIST)/index.html || $(MAKE) web
	go run .

dev: ## Запустить API и Vite dev-сервер вместе
	@trap 'kill 0' EXIT; (cd $(WEB) && npm run dev) & go run .

test: ## Прогнать тесты (Go + каталоги локалей)
	go test ./...
	cd $(WEB) && npm run check:locales

vet: ## Проверить код через go vet
	go vet ./...

fmt: ## Отформатировать Go-код
	gofmt -w .

tidy: ## Привести go.mod в порядок
	go mod tidy

clean: ## Удалить артефакты сборки
	rm -rf $(BIN) $(DIST) $(WEB)/node_modules

db-reset: ## Удалить локальную базу SQLite
	rm -f taskmanager.db taskmanager.db-shm taskmanager.db-wal

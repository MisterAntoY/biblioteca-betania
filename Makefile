# Makefile para BiblioTânia — Biblioteca Digital do Colégio Betânia

.PHONY: all build run test clean docker help

BINARY_NAME=bin/bibliotania
MAIN_PATH=./cmd/server

all: build

build:
	@echo "==> Compilando binário único otimizado..."
	@mkdir -p bin
	go build -ldflags="-s -w" -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "==> Binário gerado em $(BINARY_NAME) com sucesso!"

run: build
	@echo "==> Iniciando BiblioTânia..."
	./$(BINARY_NAME)

test:
	@echo "==> Executando suíte completa de testes de segurança e concorrência..."
	go test -v ./test/...

test-race:
	@echo "==> Executando testes com detetor de race conditions (-race)..."
	go test -race ./test/...

clean:
	@echo "==> Limpando binários e dados temporários..."
	rm -rf bin/ data/*.db data/*.db-wal data/*.db-shm

help:
	@echo "Comandos disponíveis:"
	@echo "  make build      - Compila o binário estático auto-contido"
	@echo "  make run        - Compila e executa o servidor localmente"
	@echo "  make test       - Executa todos os testes unitários e de integração"
	@echo "  make test-race  - Executa testes com detetor de race condition"
	@echo "  make clean      - Remove binários compilados"

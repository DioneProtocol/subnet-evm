# Изменения минимальной версии Golang также должны быть отражены в
# scripts/build_odyssey.sh
# Dockerfile (здесь)
# README.md
# go.mod
# ============= Стадия Компиляции ================
FROM golang:1.20.8-bullseye AS builder

WORKDIR /build
# Скопировать и скачать зависимости Odyssey используя go mod
COPY go.mod .
COPY go.sum .
RUN go mod download

# Скопировать код в контейнер
COPY . .

# Сборка odysseygo
RUN ./scripts/build.sh

# ============= Стадия Очистки ================
FROM debian:11-slim AS execution

# Поддержание совместимости с предыдущими образами
RUN mkdir -p /subnet-evm/build
WORKDIR /subnet-evm/build

# Копирование исполняемых файлов в контейнер
COPY --from=builder /build/build/ .

CMD [ "./subnet-evm" ]

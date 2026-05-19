# syntax=docker/dockerfile:1.7
# Multi-stage build: компилируем статический бинарь, кладём в distroless.

FROM golang:1.23-alpine AS build
WORKDIR /src

# Зависимости отдельным слоем (для кэша)
COPY go.mod go.sum ./
RUN go mod download

# Исходники
COPY cmd ./cmd
COPY internal ./internal

# Статическая сборка без CGO
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags='-s -w' -o /out/server ./cmd/server

# Миграции — копируем рядом с бинарём
COPY db /out/db

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/server /app/server
COPY --from=build /out/db /app/db

EXPOSE 8086
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]

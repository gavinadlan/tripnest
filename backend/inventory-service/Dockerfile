FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY inventory-service/go.mod inventory-service/go.mod
COPY common/go.mod common/go.mod

WORKDIR /app/inventory-service
RUN go mod download

WORKDIR /app
COPY common common
COPY inventory-service inventory-service

WORKDIR /app/inventory-service
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o inventory-service ./cmd/api

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/inventory-service/inventory-service .
COPY --from=builder /app/inventory-service/migrations ./migrations

EXPOSE 8084
CMD ["./inventory-service"]

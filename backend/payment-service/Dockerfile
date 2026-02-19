FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY payment-service/go.mod payment-service/go.mod
COPY payment-service/go.sum payment-service/go.sum
COPY common/go.mod common/go.mod

WORKDIR /app/payment-service
RUN go mod download

WORKDIR /app
COPY common common
COPY payment-service payment-service

WORKDIR /app/payment-service
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o payment-service ./cmd/api

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/payment-service/payment-service .
COPY --from=builder /app/payment-service/migrations ./migrations

EXPOSE 8082
CMD ["./payment-service"]

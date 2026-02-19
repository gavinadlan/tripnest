FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY search-service/go.mod search-service/go.mod
# COPY search-service/go.sum search-service/go.sum (not created yet, will be created by go mod tidy)
COPY common/go.mod common/go.mod

WORKDIR /app/search-service
RUN go mod download

WORKDIR /app
COPY common common
COPY search-service search-service

WORKDIR /app/search-service
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o search-service ./cmd/api

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/search-service/search-service .

EXPOSE 8083
CMD ["./search-service"]

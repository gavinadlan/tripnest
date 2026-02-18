FROM golang:1.24-alpine AS builder

WORKDIR /app

# 1. Copy go.mod/sum for caching
COPY user-service/go.mod user-service/go.mod
COPY user-service/go.sum user-service/go.sum
COPY common/go.mod common/go.mod

# 2. Download dependencies
WORKDIR /app/user-service
RUN go mod download

# 3. Copy source code
WORKDIR /app
COPY common common
COPY user-service user-service

# 4. Build
WORKDIR /app/user-service
# Run go mod tidy to fix any platform-specific dependency issues (like missing linux entries in go.sum)
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o user-service ./cmd/api

# Final Stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/user-service/user-service .
COPY --from=builder /app/user-service/migrations ./migrations
EXPOSE 8080
CMD ["./user-service"]
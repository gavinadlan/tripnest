FROM golang:1.24-alpine AS builder

WORKDIR /app

# 1. Copy go.mod/sum for caching
COPY booking-service/go.mod booking-service/go.mod
COPY booking-service/go.sum booking-service/go.sum
COPY common/go.mod common/go.mod

# 2. Download dependencies
WORKDIR /app/booking-service
RUN go mod download

# 3. Copy source code
WORKDIR /app
COPY common common
COPY booking-service booking-service

# 4. Build
WORKDIR /app/booking-service
# Run go mod tidy to fix any platform-specific dependency issues
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o booking-service ./cmd/api

# Final Stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/booking-service/booking-service .
# Copy migrations
COPY --from=builder /app/booking-service/migrations ./migrations
EXPOSE 8081
CMD ["./booking-service"]
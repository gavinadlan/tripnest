module github.com/gavinadlan/tripnest/backend/payment-service

go 1.23.4

require (
	github.com/gavinadlan/tripnest/backend/common v0.0.0
	github.com/golang-migrate/migrate/v4 v4.18.1
	github.com/jackc/pgx/v5 v5.7.2
	github.com/joho/godotenv v1.5.1
	github.com/segmentio/kafka-go v0.4.48
)

require (
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.15.11 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.16 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)

replace github.com/gavinadlan/tripnest/backend/common => ../common

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gavinadlan/tripnest/backend/common/auth"
	"github.com/gavinadlan/tripnest/backend/common/inbox"
	"github.com/gavinadlan/tripnest/backend/common/outbox"
	"github.com/gavinadlan/tripnest/backend/common/pgtx"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/config"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/db"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/handler"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/msgbroker"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/repository"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/service"
	"github.com/joho/godotenv"
	"github.com/segmentio/kafka-go"
)

const catalogConsumerGroup = "inventory-service-group"

func main() {
	godotenv.Load()
	cfg := config.Load()
	log.Printf("Catalog service DB target: %s", dbNameFromURL(cfg.DatabaseURL))

	verifier, err := auth.NewVerifier(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("Konfigurasi auth tidak valid: %v", err)
	}

	db.RunMigrations(cfg.DatabaseURL)

	pool, err := pgtx.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer pool.Close()
	repo := repository.NewPostgresRepository(pool)

	producer := msgbroker.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()

	svc := service.NewCatalogService(repo)
	h := handler.NewHandler(svc, verifier)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           corsMiddleware(cfg.CORSOrigins, mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Semua consumer memakai inbox.Consume: handler jalan di dalam
	// transaksi, processed_messages ditulis setelahnya dalam transaksi
	// yang sama, offset Kafka di-commit paling akhir.
	subscriptions := []subscription{
		{
			topic: "booking.created",
			apply: func(ctx context.Context, value []byte) error {
				var event model.BookingCreatedEvent
				if err := json.Unmarshal(value, &event); err != nil {
					return err
				}
				return svc.ReserveSlot(ctx, event)
			},
		},
		{
			topic: "payment.success",
			apply: func(ctx context.Context, value []byte) error {
				var event model.PaymentProcessedEvent
				if err := json.Unmarshal(value, &event); err != nil {
					return err
				}
				return svc.ConfirmReservation(ctx, event.BookingID)
			},
		},
		{
			topic: "payment.failed",
			apply: func(ctx context.Context, value []byte) error {
				var event model.PaymentProcessedEvent
				if err := json.Unmarshal(value, &event); err != nil {
					return err
				}
				return svc.ReleaseReservation(ctx, event.BookingID)
			},
		},
		{
			topic: "booking.expired",
			apply: func(ctx context.Context, value []byte) error {
				var event model.BookingExpiredEvent
				if err := json.Unmarshal(value, &event); err != nil {
					return err
				}
				return svc.ReleaseReservation(ctx, event.BookingID)
			},
		},
		{
			// Pembatalan eksplisit melepas kursi walau reservasinya sudah
			// CONFIRMED. Tanpa cabang ini, kursi booking yang dibatalkan
			// hilang dari peredaran selamanya.
			topic: "booking.cancelled",
			apply: func(ctx context.Context, value []byte) error {
				var event model.BookingCancelledEvent
				if err := json.Unmarshal(value, &event); err != nil {
					return err
				}
				return svc.ReleaseCancelled(ctx, event.BookingID)
			},
		},
	}

	for _, sub := range subscriptions {
		consumer := msgbroker.NewConsumer(cfg.KafkaBrokers, sub.topic, catalogConsumerGroup)
		defer consumer.Close()
		log.Printf("Listening for %s events...", sub.topic)
		go inbox.Consume(ctx, consumer, pool, catalogConsumerGroup, sub.handler())
	}

	go outbox.NewWorker(pool, producer, outbox.ConfigFromEnv()).Run(ctx)

	go func() {
		log.Printf("Catalog service API running on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("catalog service HTTP error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("failed to shutdown catalog HTTP server: %v", err)
	}
}

type subscription struct {
	topic string
	apply func(ctx context.Context, value []byte) error
}

// handler mengubah apply menjadi inbox.Handler. Payload yang tidak bisa
// di-decode dilaporkan sebagai kegagalan permanen supaya tidak menyumbat
// partisi; error lain diulang oleh inbox.
func (s subscription) handler() inbox.Handler {
	return func(ctx context.Context, msg kafka.Message) error {
		if err := s.apply(ctx, msg.Value); err != nil {
			var syntaxErr *json.SyntaxError
			var typeErr *json.UnmarshalTypeError
			if errors.As(err, &syntaxErr) || errors.As(err, &typeErr) {
				return inbox.Permanent(fmt.Errorf("payload %s rusak: %w", s.topic, err))
			}
			return err
		}
		return nil
	}
}

func dbNameFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "unknown"
	}
	return strings.TrimPrefix(parsed.Path, "/")
}

func corsMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

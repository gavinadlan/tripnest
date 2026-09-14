package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gavinadlan/tripnest/backend/common/auth"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/config"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/db"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/handler"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/msgbroker"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/repository"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/service"
	"github.com/joho/godotenv"
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

	repo, err := repository.NewPostgresRepository(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer repo.Close()

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

	// Keempat consumer memakai alur yang sama: baca, tandai sudah
	// diproses, decode, jalankan. Sebelumnya blok ini disalin empat kali
	// di file yang sama.
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
		go consume(ctx, consumer, repo, sub)
	}

	go runOutboxWorker(ctx, repo, producer)

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

func consume(ctx context.Context, consumer *msgbroker.Consumer, repo repository.InventoryRepository, sub subscription) {
	log.Printf("Listening for %s events...", sub.topic)
	for {
		msg, err := consumer.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			// Jangan berhenti: gangguan Kafka sesaat tidak boleh mematikan
			// consumer sampai container di-restart.
			log.Printf("%s consumer error: %v", sub.topic, err)
			continue
		}

		processed, err := repo.MarkMessageProcessed(ctx, catalogConsumerGroup, msg.Topic, string(msg.Key))
		if err != nil {
			log.Printf("failed to mark %s message: %v", sub.topic, err)
			continue
		}
		if !processed {
			continue
		}

		if err := sub.apply(ctx, msg.Value); err != nil {
			log.Printf("failed to apply %s key=%s: %v", sub.topic, string(msg.Key), err)
		}
	}
}

func runOutboxWorker(ctx context.Context, repo repository.InventoryRepository, producer *msgbroker.Producer) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			outboxEvents, err := repo.FetchPendingOutboxEvents(ctx, 50)
			if err != nil {
				log.Printf("Failed to fetch outbox events: %v", err)
				continue
			}
			for _, outboxEvent := range outboxEvents {
				var payload map[string]interface{}
				if err := json.Unmarshal(outboxEvent.Payload, &payload); err != nil {
					_ = repo.MarkOutboxEventFailed(ctx, outboxEvent.ID, err.Error())
					continue
				}
				if err := producer.Publish(ctx, outboxEvent.Topic, outboxEvent.Key, payload); err != nil {
					_ = repo.MarkOutboxEventFailed(ctx, outboxEvent.ID, err.Error())
					continue
				}
				if err := repo.MarkOutboxEventPublished(ctx, outboxEvent.ID); err != nil {
					log.Printf("failed to mark outbox published: %v", err)
					continue
				}
				log.Printf("Published outbox event topic=%s key=%s", outboxEvent.Topic, outboxEvent.Key)
			}
		}
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

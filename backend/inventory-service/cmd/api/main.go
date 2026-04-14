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

	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/config"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/db"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/handler"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/msgbroker"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/repository"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/service"
	"github.com/joho/godotenv"
)

const inventoryConsumerGroup = "inventory-service-group"

func main() {
	godotenv.Load()
	cfg := config.Load()
	log.Printf("Inventory service DB target: %s", dbNameFromURL(cfg.DatabaseURL))

	db.RunMigrations(cfg.DatabaseURL)

	repo, err := repository.NewPostgresRepository(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer repo.Close()

	producer := msgbroker.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()

	svc := service.NewInventoryService(repo, producer)
	h := handler.NewHandler(svc)

	bookingCreatedConsumer := msgbroker.NewConsumer(cfg.KafkaBrokers, "booking.created", inventoryConsumerGroup)
	defer bookingCreatedConsumer.Close()
	paymentSuccessConsumer := msgbroker.NewConsumer(cfg.KafkaBrokers, "payment.success", inventoryConsumerGroup)
	defer paymentSuccessConsumer.Close()
	paymentFailedConsumer := msgbroker.NewConsumer(cfg.KafkaBrokers, "payment.failed", inventoryConsumerGroup)
	defer paymentFailedConsumer.Close()
	bookingExpiredConsumer := msgbroker.NewConsumer(cfg.KafkaBrokers, "booking.expired", inventoryConsumerGroup)
	defer bookingExpiredConsumer.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: corsMiddleware(mux),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			msg, err := bookingCreatedConsumer.ReadMessage(ctx)
			if err != nil {
				log.Printf("booking.created consumer error: %v", err)
				break
			}
			processed, err := repo.MarkMessageProcessed(ctx, inventoryConsumerGroup, msg.Topic, string(msg.Key))
			if err != nil {
				log.Printf("failed to mark booking.created message: %v", err)
				continue
			}
			if !processed {
				continue
			}

			var event model.BookingCreatedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("invalid booking.created payload: %v", err)
				continue
			}
			if err := svc.ReserveSlot(ctx, event); err != nil {
				log.Printf("failed to reserve inventory for booking %s: %v", event.BookingID, err)
			}
		}
	}()

	go func() {
		for {
			msg, err := paymentSuccessConsumer.ReadMessage(ctx)
			if err != nil {
				log.Printf("payment.success consumer error: %v", err)
				break
			}
			processed, err := repo.MarkMessageProcessed(ctx, inventoryConsumerGroup, msg.Topic, string(msg.Key))
			if err != nil {
				log.Printf("failed to mark payment.success message: %v", err)
				continue
			}
			if !processed {
				continue
			}

			var event model.PaymentProcessedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("invalid payment.success payload: %v", err)
				continue
			}
			if err := svc.ConfirmReservation(ctx, event.BookingID); err != nil {
				log.Printf("failed to confirm reservation for booking %s: %v", event.BookingID, err)
			}
		}
	}()

	go func() {
		for {
			msg, err := paymentFailedConsumer.ReadMessage(ctx)
			if err != nil {
				log.Printf("payment.failed consumer error: %v", err)
				break
			}
			processed, err := repo.MarkMessageProcessed(ctx, inventoryConsumerGroup, msg.Topic, string(msg.Key))
			if err != nil {
				log.Printf("failed to mark payment.failed message: %v", err)
				continue
			}
			if !processed {
				continue
			}

			var event model.PaymentProcessedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("invalid payment.failed payload: %v", err)
				continue
			}
			if err := svc.ReleaseReservation(ctx, event.BookingID); err != nil {
				log.Printf("failed to release reservation for booking %s: %v", event.BookingID, err)
			}
		}
	}()

	go func() {
		for {
			msg, err := bookingExpiredConsumer.ReadMessage(ctx)
			if err != nil {
				log.Printf("booking.expired consumer error: %v", err)
				break
			}
			processed, err := repo.MarkMessageProcessed(ctx, inventoryConsumerGroup, msg.Topic, string(msg.Key))
			if err != nil {
				log.Printf("failed to mark booking.expired message: %v", err)
				continue
			}
			if !processed {
				continue
			}

			var event model.BookingExpiredEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("invalid booking.expired payload: %v", err)
				continue
			}
			if err := svc.ReleaseReservation(ctx, event.BookingID); err != nil {
				log.Printf("failed to release expired reservation for booking %s: %v", event.BookingID, err)
			}
		}
	}()

	go func() {
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
	}()

	go func() {
		log.Printf("Inventory service API running on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("inventory service HTTP error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("failed to shutdown inventory HTTP server: %v", err)
	}
}

func dbNameFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "unknown"
	}
	return strings.TrimPrefix(parsed.Path, "/")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
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

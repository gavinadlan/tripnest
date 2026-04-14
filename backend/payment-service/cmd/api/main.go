package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gavinadlan/tripnest/backend/payment-service/internal/config"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/db"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/handler"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/midtrans"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/msgbroker"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/repository"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/service"
	"github.com/joho/godotenv"
)

const paymentConsumerGroup = "payment-service-group"

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file loaded: %v", err)
	}
	cfg := config.Load()
	log.Printf("Payment service DB target: %s", dbNameFromURL(cfg.DatabaseURL))
	log.Printf("BOOKING_SERVICE_URL=%s", cfg.BookingURL)
	log.Printf("MIDTRANS_SERVER_KEY loaded=%t", cfg.Midtrans.ServerKey != "")
	log.Printf("MIDTRANS_CLIENT_KEY loaded=%t", cfg.Midtrans.ClientKey != "")
	if cfg.Midtrans.ServerKey == "" || cfg.Midtrans.ClientKey == "" {
		log.Fatal(errors.New("missing MIDTRANS_SERVER_KEY or MIDTRANS_CLIENT_KEY"))
	}

	db.RunMigrations(cfg.DatabaseURL)

	repo, err := repository.NewPostgresRepository(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer repo.Close()

	producer := msgbroker.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()

	midtransClient := midtrans.NewClient(cfg.Midtrans.ServerKey, cfg.Midtrans.IsProduction)
	svc := service.NewPaymentService(repo, producer, midtransClient, cfg.Midtrans.ServerKey, cfg.BookingURL)
	h := handler.NewHandler(svc)

	consumer := msgbroker.NewConsumer(cfg.KafkaBrokers, "booking.created", paymentConsumerGroup)
	defer consumer.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: corsMiddleware(mux),
	}

	log.Println("Payment Service Started (Listening for booking.created)")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			msg, err := consumer.ReadMessage(ctx)
			if err != nil {
				log.Printf("Consumer error: %v", err)
				break
			}

			processed, err := repo.MarkMessageProcessed(ctx, paymentConsumerGroup, msg.Topic, string(msg.Key))
			if err != nil {
				log.Printf("Failed to mark booking.created message: %v", err)
				continue
			}
			if !processed {
				continue
			}

			var event model.BookingEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Failed to unmarshal event: %v", err)
				continue
			}

			log.Printf("Received booking event: %v", event.BookingID)
			go svc.ProcessPayment(ctx, event)
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
					if strings.TrimSpace(outboxEvent.Key) == "" {
						_ = repo.MarkOutboxEventFailed(ctx, outboxEvent.ID, "empty message key")
						log.Printf("Skipping outbox event with empty key topic=%s id=%s", outboxEvent.Topic, outboxEvent.ID)
						continue
					}
					var payload map[string]interface{}
					if err := json.Unmarshal(outboxEvent.Payload, &payload); err != nil {
						_ = repo.MarkOutboxEventFailed(ctx, outboxEvent.ID, err.Error())
						continue
					}
					log.Printf("Publishing event to Kafka topic=%s key=%s", outboxEvent.Topic, outboxEvent.Key)
					if err := producer.Publish(ctx, outboxEvent.Topic, outboxEvent.Key, payload); err != nil {
						_ = repo.MarkOutboxEventFailed(ctx, outboxEvent.ID, err.Error())
						continue
					}
					if err := repo.MarkOutboxEventPublished(ctx, outboxEvent.ID); err != nil {
						log.Printf("Failed to mark outbox event published: %v", err)
						continue
					}
					log.Printf("Published outbox event topic=%s key=%s", outboxEvent.Topic, outboxEvent.Key)
				}
			}
		}
	}()

	go func() {
		log.Printf("Payment service API running on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Failed to shutdown HTTP server: %v", err)
	}

	log.Println("Shutting down Payment Service")
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

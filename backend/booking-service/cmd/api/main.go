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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	"github.com/gavinadlan/tripnest/backend/booking-service/internal/config"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/db"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/events"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/handler"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/repository"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/service"
)

const bookingConsumerGroup = "booking-service-group"

func main() {
	godotenv.Load()
	cfg := config.Load()
	log.Printf("Booking service DB target: %s", dbNameFromURL(cfg.DatabaseURL))

	// Run migrations
	db.RunMigrations(cfg.DatabaseURL)

	repo, err := repository.NewPostgresRepository(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer repo.Close()

	producer := events.NewKafkaProducer(cfg.KafkaBrokers)
	defer producer.Close()

	svc := service.NewBookingService(repo, producer, cfg.BookingExpiryMinutes)

	// Payment Success Consumer
	paymentSuccessConsumer := events.NewConsumer(cfg.KafkaBrokers, "payment.success", bookingConsumerGroup)
	defer paymentSuccessConsumer.Close()

	// Payment Failed Consumer
	paymentFailedConsumer := events.NewConsumer(cfg.KafkaBrokers, "payment.failed", bookingConsumerGroup)
	defer paymentFailedConsumer.Close()

	h := handler.NewHandler(svc)

	r := chi.NewRouter()

	// CORS Setup
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, // Frontend URL
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	h.RegisterRoutes(r)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Start consumers safely
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Println("Listening for payment.success events...")
		for {
			msg, err := paymentSuccessConsumer.ReadMessage(ctx)
			if err != nil {
				log.Printf("Payment Success Consumer error: %v", err)
				break
			}

			processed, err := repo.MarkMessageProcessed(ctx, bookingConsumerGroup, msg.Topic, string(msg.Key))
			if err != nil {
				log.Printf("Failed to mark message processed: %v", err)
				continue
			}
			if !processed {
				continue
			}

			var event model.PaymentProcessedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Failed to unmarshal payment success event: %v", err)
				continue
			}

			if err := svc.ConfirmBooking(ctx, event.BookingID); err != nil {
				log.Printf("Failed to confirm booking %s: %v", event.BookingID, err)
			}
		}
	}()

	go func() {
		log.Println("Listening for payment.failed events...")
		for {
			msg, err := paymentFailedConsumer.ReadMessage(ctx)
			if err != nil {
				log.Printf("Payment Failed Consumer error: %v", err)
				break
			}

			processed, err := repo.MarkMessageProcessed(ctx, bookingConsumerGroup, msg.Topic, string(msg.Key))
			if err != nil {
				log.Printf("Failed to mark message processed: %v", err)
				continue
			}
			if !processed {
				continue
			}

			var event model.PaymentProcessedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Failed to unmarshal payment failed event: %v", err)
				continue
			}

			if err := svc.CancelBooking(ctx, event.BookingID); err != nil {
				log.Printf("Failed to cancel booking %s: %v", event.BookingID, err)
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
						log.Printf("Failed to mark outbox event published: %v", err)
						continue
					}
					log.Printf("Published outbox event topic=%s key=%s", outboxEvent.Topic, outboxEvent.Key)
				}
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(time.Duration(cfg.BookingExpiryInterval) * time.Second)
		defer ticker.Stop()
		log.Printf("Starting booking expiration worker (interval=%ds)", cfg.BookingExpiryInterval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := svc.ExpirePendingBookings(ctx); err != nil {
					log.Printf("Failed to expire pending bookings: %v", err)
				}
			}
		}
	}()

	go func() {
		log.Printf("Booking Service starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}

func dbNameFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "unknown"
	}
	return strings.TrimPrefix(parsed.Path, "/")
}

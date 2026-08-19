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

	"github.com/gavinadlan/tripnest/backend/booking-service/internal/catalog"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/config"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/db"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/events"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/handler"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/repository"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/service"
	"github.com/gavinadlan/tripnest/backend/common/auth"
)

const (
	paymentSuccessConsumerGroup = "booking-service-group-payment-success"
	paymentFailedConsumerGroup  = "booking-service-group-payment-failed"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	log.Printf("Booking service DB target: %s", dbNameFromURL(cfg.DatabaseURL))

	verifier, err := auth.NewVerifier(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("Konfigurasi auth tidak valid: %v", err)
	}

	db.RunMigrations(cfg.DatabaseURL)

	repo, err := repository.NewPostgresRepository(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer repo.Close()

	producer := events.NewKafkaProducer(cfg.KafkaBrokers)
	defer producer.Close()

	catalogClient := catalog.New(cfg.CatalogURL)
	log.Printf("Sumber harga trip: %s", cfg.CatalogURL)

	svc := service.NewBookingService(repo, catalogClient, cfg.BookingExpiryMinutes)

	paymentSuccessConsumer := events.NewConsumer(cfg.KafkaBrokers, "payment.success", paymentSuccessConsumerGroup)
	defer paymentSuccessConsumer.Close()

	paymentFailedConsumer := events.NewConsumer(cfg.KafkaBrokers, "payment.failed", paymentFailedConsumerGroup)
	defer paymentFailedConsumer.Close()

	h := handler.NewHandler(svc)

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", h.Health)

	// Seluruh endpoint booking wajib token. Identitas pemanggil diambil
	// dari klaim, tidak lagi dari body request.
	r.Group(func(r chi.Router) {
		r.Use(verifier.Required)
		h.RegisterRoutes(r)
	})

	// Endpoint internal untuk payment-service: membaca booking apa pun
	// tanpa cek kepemilikan, tapi hanya untuk token bertipe service/admin.
	r.Group(func(r chi.Router) {
		r.Use(verifier.RequireServiceOrAdmin)
		r.Get("/internal/bookings/{id}", h.GetBookingInternal)
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go consumePaymentOutcome(ctx, paymentSuccessConsumer, repo, paymentSuccessConsumerGroup, "payment.success", svc.ConfirmBooking)
	go consumePaymentOutcome(ctx, paymentFailedConsumer, repo, paymentFailedConsumerGroup, "payment.failed", svc.CancelBooking)

	go runOutboxWorker(ctx, repo, producer)
	go runExpiryWorker(ctx, svc, cfg.BookingExpiryInterval)

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

// consumePaymentOutcome menangani payment.success dan payment.failed, yang
// alurnya identik kecuali aksi akhirnya. Sebelumnya dua blok ini disalin
// hampir sama persis di main.
func consumePaymentOutcome(
	ctx context.Context,
	consumer *events.Consumer,
	repo repository.BookingRepository,
	consumerGroup string,
	topic string,
	apply func(context.Context, string) error,
) {
	log.Printf("Listening for %s events...", topic)
	for {
		msg, err := consumer.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("%s consumer error: %v", topic, err)
			continue
		}

		processed, err := repo.MarkMessageProcessed(ctx, consumerGroup, msg.Topic, string(msg.Key))
		if err != nil {
			log.Printf("Failed to mark message processed: %v", err)
			continue
		}
		if !processed {
			continue
		}

		var event model.PaymentProcessedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Failed to unmarshal %s event: %v", topic, err)
			continue
		}

		log.Printf("Received %s booking_id=%s status=%s", topic, event.BookingID, event.Status)
		if err := apply(ctx, event.BookingID); err != nil {
			log.Printf("Failed to apply %s for booking %s: %v", topic, event.BookingID, err)
		}
	}
}

func runOutboxWorker(ctx context.Context, repo repository.BookingRepository, producer *events.KafkaProducer) {
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
}

func runExpiryWorker(ctx context.Context, svc service.BookingService, intervalSeconds int) {
	interval := time.Duration(intervalSeconds) * time.Second
	if interval < time.Minute {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starting booking expiration worker (interval=%s)", interval)
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
}

func dbNameFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "unknown"
	}
	return strings.TrimPrefix(parsed.Path, "/")
}

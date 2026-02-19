package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/gavinadlan/tripnest/backend/booking-service/internal/config"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/db"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/events"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/handler"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/repository"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/service"
)

func main() {
	godotenv.Load()
	cfg := config.Load()

	// Run migrations
	db.RunMigrations(cfg.DatabaseURL)

	repo, err := repository.NewPostgresRepository(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer repo.Close()

	producer := events.NewKafkaProducer(cfg.KafkaBrokers)
	defer producer.Close()

	svc := service.NewBookingService(repo, producer)

	// Payment Success Consumer
	paymentSuccessConsumer := events.NewConsumer(cfg.KafkaBrokers, "payment.success", "booking-service-group")
	defer paymentSuccessConsumer.Close()

	// Payment Failed Consumer
	paymentFailedConsumer := events.NewConsumer(cfg.KafkaBrokers, "payment.failed", "booking-service-group")
	defer paymentFailedConsumer.Close()

	h := handler.NewHandler(svc)

	r := chi.NewRouter()
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

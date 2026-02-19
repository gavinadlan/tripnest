package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gavinadlan/tripnest/backend/payment-service/internal/config"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/db"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/msgbroker"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/repository"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/service"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	cfg := config.Load()

	db.RunMigrations(cfg.DatabaseURL)

	repo, err := repository.NewPostgresRepository(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer repo.Close()

	producer := msgbroker.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()

	svc := service.NewPaymentService(repo, producer)

	consumer := msgbroker.NewConsumer(cfg.KafkaBrokers, "booking.created", "payment-service-group")
	defer consumer.Close()

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

			var event model.BookingEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Failed to unmarshal event: %v", err)
				continue
			}

			log.Printf("Received booking event: %v", event.BookingID)
			go svc.ProcessPayment(ctx, event)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Payment Service")
}

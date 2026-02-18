package main

import (
	"context"
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
	h := handler.NewHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	h.RegisterRoutes(r)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}

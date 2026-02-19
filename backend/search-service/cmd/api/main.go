package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gavinadlan/tripnest/backend/search-service/internal/config"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/handler"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/repository"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	cfg := config.Load()

	log.Printf("Starting Search Service on port %s", cfg.Port)

	// Initialize Mongo Repository
	mongoRepo, err := repository.NewMongoRepository(cfg.MongoURI, "tripnest_search")
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Wrap with Redis Cache
	cachedRepo := repository.NewCachedListingRepository(mongoRepo, cfg.RedisAddr, 60*time.Second)

	// Initialize Service
	svc := service.NewSearchService(cachedRepo) // cachedRepo implements ListingRepository

	// Initialize Handler
	h := handler.NewHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Seed endpoint for testing
	r.Post("/seed", h.Seed)

	r.Get("/search", h.Search)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

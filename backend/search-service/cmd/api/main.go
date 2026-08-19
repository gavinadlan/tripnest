package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gavinadlan/tripnest/backend/common/auth"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/config"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/handler"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/repository"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	verifier, err := auth.NewVerifier(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("Konfigurasi auth tidak valid: %v", err)
	}

	log.Printf("Starting Search Service on port %s", cfg.Port)

	mongoRepo, err := repository.NewMongoRepository(cfg.MongoURI, "tripnest_search")
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	cachedRepo := repository.NewCachedListingRepository(mongoRepo, cfg.RedisAddr, 60*time.Second)
	svc := service.NewSearchService(cachedRepo)
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

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Katalog boleh dibaca siapa saja: ini etalase publik.
	r.Get("/search", h.Search)
	r.Get("/trips", h.ListTrips)
	r.Get("/trips/{id}", h.GetTrip)

	// Menulis katalog hanya untuk admin. Sebelumnya endpoint ini terbuka,
	// jadi siapa pun bisa menghapus seluruh trip.
	r.Group(func(r chi.Router) {
		r.Use(verifier.Required)
		r.Use(requireAdmin)

		r.Post("/trips", h.CreateTrip)
		r.Put("/trips/{id}", h.UpdateTrip)
		r.Delete("/trips/{id}", h.DeleteTrip)
		r.Post("/seed", h.Seed)
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
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
	_ = server.Shutdown(ctx)
}

// requireAdmin dipasang setelah verifier.Required, sehingga claims sudah
// pasti ada di context saat sampai di sini.
func requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !auth.IsAdmin(r.Context()) {
			http.Error(w, `{"error":"akses ditolak"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

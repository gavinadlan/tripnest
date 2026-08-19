package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/gavinadlan/tripnest/backend/common/auth"
	"github.com/gavinadlan/tripnest/backend/common/ratelimit"
	"github.com/gavinadlan/tripnest/backend/user-service/internal/config"
	"github.com/gavinadlan/tripnest/backend/user-service/internal/db"
	"github.com/gavinadlan/tripnest/backend/user-service/internal/handler"
	"github.com/gavinadlan/tripnest/backend/user-service/internal/repository"
	"github.com/gavinadlan/tripnest/backend/user-service/internal/service"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	// Gagal cepat kalau secret lemah: service yang jalan dengan secret
	// bisa-ditebak lebih berbahaya daripada service yang tidak jalan.
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

	svc := service.NewUserService(repo, cfg, verifier)
	h := handler.NewHandler(svc)

	if cfg.AdminEmail != "" && cfg.AdminPassword != "" {
		bootstrapCtx, cancelBootstrap := context.WithTimeout(context.Background(), 10*time.Second)
		if err := svc.EnsureAdmin(bootstrapCtx, cfg.AdminEmail, cfg.AdminPassword); err != nil {
			log.Fatalf("Gagal menyiapkan akun admin: %v", err)
		}
		log.Printf("Akun admin siap: %s", cfg.AdminEmail)
		cancelBootstrap()
	} else {
		log.Println("ADMIN_EMAIL/ADMIN_PASSWORD tidak diisi, bootstrap admin dilewati")
	}

	// Brute force di /login dan /register dibatasi per alamat IP.
	authLimiter := ratelimit.New(10, time.Minute)
	defer authLimiter.Close()

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", h.Health)

	r.Group(func(r chi.Router) {
		r.Use(authLimiter.Middleware)
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
	})

	r.Group(func(r chi.Router) {
		r.Use(verifier.Required)
		r.Get("/me", h.Me)
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("User Service starting on port %s", cfg.Port)
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

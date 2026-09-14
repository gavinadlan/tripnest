package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gavinadlan/tripnest/backend/common/auth"
	"github.com/gavinadlan/tripnest/backend/common/inbox"
	"github.com/gavinadlan/tripnest/backend/common/outbox"
	"github.com/gavinadlan/tripnest/backend/common/pgtx"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/config"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/db"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/handler"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/midtrans"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/msgbroker"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/repository"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/service"
	"github.com/joho/godotenv"
	"github.com/segmentio/kafka-go"
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

	verifier, err := auth.NewVerifier(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("Konfigurasi auth tidak valid: %v", err)
	}

	db.RunMigrations(cfg.DatabaseURL)

	pool, err := pgtx.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer pool.Close()
	repo := repository.NewPostgresRepository(pool)

	producer := msgbroker.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()

	midtransClient := midtrans.NewClient(cfg.Midtrans.ServerKey, cfg.Midtrans.IsProduction)
	svc := service.NewPaymentService(repo, producer, midtransClient, cfg.Midtrans.ServerKey, cfg.BookingURL, verifier)
	h := handler.NewHandler(svc, verifier)

	// Pemicunya inventory.reserved, bukan booking.created: transaksi Snap
	// hanya boleh dibuat setelah kursinya benar-benar tertahan. Dulu
	// booking.created langsung memicu pembuatan token, jadi booking yang
	// kalah balapan kursi pun sempat punya token dan bisa dibayar.
	consumer := msgbroker.NewConsumer(cfg.KafkaBrokers, "inventory.reserved", paymentConsumerGroup)
	defer consumer.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           corsMiddleware(cfg.CORSOrigins, mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Println("Payment Service Started (Listening for inventory.reserved)")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handler jalan di dalam transaksi yang dibuka inbox.Consume; catatan
	// processed_messages ditulis setelahnya, dalam transaksi yang sama.
	// Kalau Midtrans atau database sedang tidak bisa dihubungi, pesan
	// diulang di tempat -- tidak hilang, tidak dilewati.
	go inbox.Consume(ctx, consumer, pool, paymentConsumerGroup, func(ctx context.Context, msg kafka.Message) error {
		var event model.BookingEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return inbox.Permanent(fmt.Errorf("payload inventory.reserved rusak: %w", err))
		}
		log.Printf("Kursi tertahan, menyiapkan pembayaran booking_id=%s", event.BookingID)
		return svc.ProcessPayment(ctx, event)
	})

	go outbox.NewWorker(pool, producer, outbox.ConfigFromEnv()).Run(ctx)

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

func corsMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
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

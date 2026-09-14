package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/segmentio/kafka-go"

	"github.com/gavinadlan/tripnest/backend/booking-service/internal/catalog"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/config"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/db"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/events"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/handler"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/repository"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/service"
	"github.com/gavinadlan/tripnest/backend/common/auth"
	"github.com/gavinadlan/tripnest/backend/common/inbox"
	"github.com/gavinadlan/tripnest/backend/common/outbox"
	"github.com/gavinadlan/tripnest/backend/common/pgtx"
)

const (
	paymentSuccessConsumerGroup = "booking-service-group-payment-success"
	paymentFailedConsumerGroup  = "booking-service-group-payment-failed"
	inventoryRejectedGroup      = "booking-service-group-inventory-rejected"
	inventoryReservedGroup      = "booking-service-group-inventory-reserved"
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

	pool, err := pgtx.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()
	repo := repository.NewPostgresRepository(pool)

	producer := events.NewKafkaProducer(cfg.KafkaBrokers)
	defer producer.Close()

	catalogClient := catalog.New(cfg.CatalogURL, verifier)
	log.Printf("Sumber harga trip: %s", cfg.CatalogURL)

	svc := service.NewBookingService(repo, catalogClient, cfg.BookingExpiryMinutes, cfg.SeatTimeoutSeconds)

	paymentSuccessConsumer := events.NewConsumer(cfg.KafkaBrokers, "payment.success", paymentSuccessConsumerGroup)
	defer paymentSuccessConsumer.Close()

	paymentFailedConsumer := events.NewConsumer(cfg.KafkaBrokers, "payment.failed", paymentFailedConsumerGroup)
	defer paymentFailedConsumer.Close()

	// Penolakan kursi dari catalog-service. Tanpa consumer ini, booking
	// yang kursinya sudah habis tetap menunggu pembayaran sampai
	// kedaluwarsa -- kursi yang tidak ada tetap terlihat bisa dibayar.
	inventoryRejectedConsumer := events.NewConsumer(cfg.KafkaBrokers, "inventory.rejected", inventoryRejectedGroup)
	defer inventoryRejectedConsumer.Close()

	// Penahanan kursi yang berhasil. Inilah yang memindahkan booking dari
	// PENDING_SEAT ke PENDING_PAYMENT -- satu-satunya jalan menuju
	// pembayaran.
	inventoryReservedConsumer := events.NewConsumer(cfg.KafkaBrokers, "inventory.reserved", inventoryReservedGroup)
	defer inventoryReservedConsumer.Close()

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

	// Setiap handler jalan di dalam transaksi yang dibuka inbox.Consume;
	// catatan processed_messages ditulis setelah handler, dalam transaksi
	// yang sama. Handler yang gagal diulang, bukan dilewati.
	go inbox.Consume(ctx, paymentSuccessConsumer, pool, paymentSuccessConsumerGroup,
		decode(func(ctx context.Context, event model.PaymentProcessedEvent) error {
			log.Printf("Received payment.success booking_id=%s status=%s", event.BookingID, event.Status)
			return svc.ConfirmBooking(ctx, event)
		}))
	go inbox.Consume(ctx, paymentFailedConsumer, pool, paymentFailedConsumerGroup,
		decode(func(ctx context.Context, event model.PaymentProcessedEvent) error {
			log.Printf("Received payment.failed booking_id=%s status=%s", event.BookingID, event.Status)
			// Pembatalan tidak butuh detail pembayaran; hanya konfirmasi
			// yang perlu mencatat nominal saat uangnya jadi yatim.
			return svc.CancelBooking(ctx, event.BookingID)
		}))
	// Kursi ditolak catalog-service. Kursi tidak perlu dilepas balik:
	// reservasinya memang tidak pernah terbentuk.
	go inbox.Consume(ctx, inventoryRejectedConsumer, pool, inventoryRejectedGroup,
		decode(func(ctx context.Context, event model.InventoryRejectedEvent) error {
			log.Printf("Kursi ditolak booking_id=%s resource_id=%s reason=%s", event.BookingID, event.ResourceID, event.Reason)
			return svc.CancelUnseatedBooking(ctx, event.BookingID)
		}))
	// Kursi tertahan: satu-satunya jalan dari PENDING_SEAT ke
	// PENDING_PAYMENT. payment-service mendengarkan topik yang sama.
	go inbox.Consume(ctx, inventoryReservedConsumer, pool, inventoryReservedGroup,
		decode(func(ctx context.Context, event model.InventoryReservedEvent) error {
			log.Printf("Kursi tertahan booking_id=%s resource_id=%s qty=%d", event.BookingID, event.ResourceID, event.Quantity)
			return svc.MarkSeatReserved(ctx, event)
		}))

	go outbox.NewWorker(pool, producer, outbox.ConfigFromEnv()).Run(ctx)
	go runExpiryWorker(ctx, svc, cfg.BookingExpiryInterval)
	go runStuckSeatWorker(ctx, svc)

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

// decode membungkus handler bertipe dengan pembacaan payload. Payload yang
// tidak bisa di-decode adalah kegagalan permanen: mengulangnya tidak akan
// mengubah apa pun, jadi pesannya dilewati dan dicatat.
func decode[T any](apply func(context.Context, T) error) inbox.Handler {
	return func(ctx context.Context, msg kafka.Message) error {
		var event T
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return inbox.Permanent(fmt.Errorf("payload %s rusak: %w", msg.Topic, err))
		}
		return apply(ctx, event)
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

// runStuckSeatWorker berjalan jauh lebih sering daripada worker
// kedaluwarsa: menunggu kursi diukur dalam detik, bukan menit, dan
// booking yang nyangkut di PENDING_SEAT tidak terlihat oleh worker mana
// pun kalau tidak ada yang menyapunya.
func runStuckSeatWorker(ctx context.Context, svc service.BookingService) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("Starting stuck-seat sweeper (interval=30s)")
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := svc.ExpireStuckSeatBookings(ctx); err != nil {
				log.Printf("Failed to sweep stuck seat bookings: %v", err)
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

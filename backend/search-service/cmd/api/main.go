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

	"github.com/gavinadlan/tripnest/backend/search-service/internal/config"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/handler"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/msgbroker"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/repository"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

// Satu consumer group untuk kedua topik katalog: keduanya menuju read
// model yang sama.
const catalogConsumerGroup = "search-service-group-catalog"

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

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

	// Read model hanya punya endpoint baca. Menulis katalog dilakukan di
	// catalog-service; dulu trip bisa dibuat di sini juga, dan trip hasil
	// jalur itu tidak pernah punya baris kapasitas.
	r.Get("/search", h.Search)
	r.Get("/trips", h.ListTrips)
	r.Get("/trips/{id}", h.GetTrip)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	consumerCtx, stopConsumers := context.WithCancel(context.Background())
	defer stopConsumers()

	upsertedConsumer := msgbroker.NewConsumer(cfg.KafkaBrokers, "trip.upserted", catalogConsumerGroup)
	defer upsertedConsumer.Close()
	go consume(consumerCtx, upsertedConsumer, "trip.upserted", func(ctx context.Context, value []byte) error {
		var event model.TripUpsertedEvent
		if err := json.Unmarshal(value, &event); err != nil {
			return err
		}
		return svc.ApplyTripUpserted(ctx, event)
	})

	deletedConsumer := msgbroker.NewConsumer(cfg.KafkaBrokers, "trip.deleted", catalogConsumerGroup)
	defer deletedConsumer.Close()
	go consume(consumerCtx, deletedConsumer, "trip.deleted", func(ctx context.Context, value []byte) error {
		var event model.TripDeletedEvent
		if err := json.Unmarshal(value, &event); err != nil {
			return err
		}
		return svc.ApplyTripDeleted(ctx, event)
	})

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

// consume menerapkan event ke read model. Tidak ada tabel
// processed_messages di sini: upsert dan delete keduanya idempoten, jadi
// event yang terkirim ulang aman.
func consume(ctx context.Context, consumer *msgbroker.Consumer, topic string, apply func(context.Context, []byte) error) {
	log.Printf("Listening for %s events...", topic)
	for {
		msg, err := consumer.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			// Gangguan Kafka sesaat tidak boleh mematikan consumer sampai
			// container di-restart.
			log.Printf("%s consumer error: %v", topic, err)
			continue
		}

		if err := apply(ctx, msg.Value); err != nil {
			log.Printf("failed to apply %s key=%s: %v", topic, string(msg.Key), err)
			continue
		}
		log.Printf("Read model diperbarui dari %s key=%s", topic, string(msg.Key))
	}
}

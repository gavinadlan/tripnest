package service

import (
	"context"
	"errors"
	"time"

	"github.com/gavinadlan/tripnest/backend/search-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/repository"
)

type SearchService interface {
	SearchListings(ctx context.Context, params *model.SearchParams) ([]*model.Listing, int64, error)
	ListTrips(ctx context.Context) ([]*model.Listing, error)
	GetTrip(ctx context.Context, id string) (*model.Listing, error)
	// ApplyTripUpserted dan ApplyTripDeleted dipanggil consumer, bukan
	// handler HTTP. Katalog ditulis di catalog-service.
	ApplyTripUpserted(ctx context.Context, event model.TripUpsertedEvent) error
	ApplyTripDeleted(ctx context.Context, event model.TripDeletedEvent) error
}

type searchService struct {
	repo repository.ListingRepository
}

func NewSearchService(repo repository.ListingRepository) SearchService {
	return &searchService{repo: repo}
}

func (s *searchService) SearchListings(ctx context.Context, params *model.SearchParams) ([]*model.Listing, int64, error) {
	return s.repo.Search(ctx, params)
}

func (s *searchService) ListTrips(ctx context.Context) ([]*model.Listing, error) {
	return s.repo.List(ctx)
}

func (s *searchService) GetTrip(ctx context.Context, id string) (*model.Listing, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *searchService) ApplyTripUpserted(ctx context.Context, event model.TripUpsertedEvent) error {
	if event.ID == "" {
		return errors.New("trip.upserted tanpa id")
	}

	// updated_at datang sebagai RFC3339 dari catalog-service. Kalau
	// bentuknya tidak terbaca, pakai waktu sekarang daripada menolak
	// event dan membiarkan read model tertinggal.
	updatedAt, err := time.Parse(time.RFC3339, event.UpdatedAt)
	if err != nil {
		updatedAt = time.Now().UTC()
	}

	return s.repo.Upsert(ctx, &model.Listing{
		ID:             event.ID,
		Title:          event.Title,
		Destination:    event.Destination,
		Price:          event.Price,
		Date:           event.Date,
		TotalSlots:     event.TotalSlots,
		AvailableSlots: event.AvailableSlots,
		UpdatedAt:      updatedAt.UTC(),
	})
}

func (s *searchService) ApplyTripDeleted(ctx context.Context, event model.TripDeletedEvent) error {
	if event.ID == "" {
		return errors.New("trip.deleted tanpa id")
	}
	return s.repo.Delete(ctx, event.ID)
}

package service

import (
	"context"

	"github.com/gavinadlan/tripnest/backend/search-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/repository"
)

type SearchService interface {
	SearchListings(ctx context.Context, params *model.SearchParams) ([]*model.Listing, int64, error)
	ListTrips(ctx context.Context) ([]*model.Listing, error)
	CreateTrip(ctx context.Context, trip *model.Listing) error
	UpdateTrip(ctx context.Context, id string, trip *model.Listing) error
	DeleteTrip(ctx context.Context, id string) error
	SeedListings(ctx context.Context) error
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

func (s *searchService) SeedListings(ctx context.Context) error {
	return s.repo.Seed(ctx)
}

func (s *searchService) ListTrips(ctx context.Context) ([]*model.Listing, error) {
	return s.repo.List(ctx)
}

func (s *searchService) CreateTrip(ctx context.Context, trip *model.Listing) error {
	return s.repo.Create(ctx, trip)
}

func (s *searchService) UpdateTrip(ctx context.Context, id string, trip *model.Listing) error {
	return s.repo.Update(ctx, id, trip)
}

func (s *searchService) DeleteTrip(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

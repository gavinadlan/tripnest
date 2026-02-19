package service

import (
	"context"

	"github.com/gavinadlan/tripnest/backend/search-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/repository"
)

type SearchService interface {
	SearchListings(ctx context.Context, params *model.SearchParams) ([]*model.Listing, int64, error)
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

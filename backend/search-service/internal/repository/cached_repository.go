package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gavinadlan/tripnest/backend/search-service/internal/model"
	"github.com/go-redis/redis/v8"
)

type CachedListingRepository struct {
	next ListingRepository
	rdb  *redis.Client
	ttl  time.Duration
}

func NewCachedListingRepository(next ListingRepository, redisAddr string, ttl time.Duration) *CachedListingRepository {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	return &CachedListingRepository{
		next: next,
		rdb:  rdb,
		ttl:  ttl,
	}
}

func (r *CachedListingRepository) Search(ctx context.Context, params *model.SearchParams) ([]*model.Listing, int64, error) {
	cacheKey := fmt.Sprintf("search:%s:%f:%f:%s:%d:%d",
		params.Destination, params.MinPrice, params.MaxPrice, params.Date, params.Page, params.Limit)

	val, err := r.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		log.Println("CACHE HIT")
		var result struct {
			Listings []*model.Listing `json:"data"`
			Total    int64            `json:"total"`
		}
		if err := json.Unmarshal([]byte(val), &result); err == nil {
			return result.Listings, result.Total, nil
		}
	} else if err != redis.Nil {
		log.Printf("Redis error: %v", err)
	}

	log.Println("CACHE MISS")
	listings, total, err := r.next.Search(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	// Cache result
	cacheData := struct {
		Listings []*model.Listing `json:"data"`
		Total    int64            `json:"total"`
	}{
		Listings: listings,
		Total:    total,
	}

	if data, err := json.Marshal(cacheData); err == nil {
		r.rdb.Set(ctx, cacheKey, data, r.ttl)
	}

	return listings, total, nil
}

func (r *CachedListingRepository) Seed(ctx context.Context) error {
	return r.next.Seed(ctx)
}

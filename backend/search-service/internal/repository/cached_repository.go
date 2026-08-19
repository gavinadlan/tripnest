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
	if err := r.next.Seed(ctx); err != nil {
		return err
	}
	r.invalidate(ctx)
	return nil
}

func (r *CachedListingRepository) GetByID(ctx context.Context, id string) (*model.Listing, error) {
	return r.next.GetByID(ctx, id)
}

// invalidate menghapus seluruh hasil search yang di-cache.
//
// Tanpa ini, perubahan trip baru terlihat setelah TTL habis dan admin
// mengira simpanannya gagal. Pola search:* jumlahnya kecil di skala ini,
// jadi SCAN sekali jalan sudah memadai.
func (r *CachedListingRepository) invalidate(ctx context.Context) {
	var cursor uint64
	for {
		keys, next, err := r.rdb.Scan(ctx, cursor, "search:*", 200).Result()
		if err != nil {
			log.Printf("Gagal memindai cache untuk invalidasi: %v", err)
			return
		}
		if len(keys) > 0 {
			if err := r.rdb.Del(ctx, keys...).Err(); err != nil {
				log.Printf("Gagal menghapus cache: %v", err)
				return
			}
		}
		if next == 0 {
			return
		}
		cursor = next
	}
}

func (r *CachedListingRepository) List(ctx context.Context) ([]*model.Listing, error) {
	return r.next.List(ctx)
}

func (r *CachedListingRepository) Create(ctx context.Context, listing *model.Listing) error {
	if err := r.next.Create(ctx, listing); err != nil {
		return err
	}
	r.invalidate(ctx)
	return nil
}

func (r *CachedListingRepository) Update(ctx context.Context, id string, listing *model.Listing) error {
	if err := r.next.Update(ctx, id, listing); err != nil {
		return err
	}
	r.invalidate(ctx)
	return nil
}

func (r *CachedListingRepository) Delete(ctx context.Context, id string) error {
	if err := r.next.Delete(ctx, id); err != nil {
		return err
	}
	r.invalidate(ctx)
	return nil
}

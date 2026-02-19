package repository

import (
	"context"
	"log"
	"time"

	"github.com/gavinadlan/tripnest/backend/search-service/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ListingRepository interface {
	Search(ctx context.Context, params *model.SearchParams) ([]*model.Listing, int64, error)
	Seed(ctx context.Context) error
}

type mongoRepository struct {
	coll *mongo.Collection
}

func NewMongoRepository(uri string, dbName string) (ListingRepository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	coll := client.Database(dbName).Collection("listings")

	// Create Indexes
	_, err = coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "destination", Value: 1}}},
		{Keys: bson.D{{Key: "price", Value: 1}}},
		{Keys: bson.D{{Key: "date", Value: 1}}},
	})
	if err != nil {
		log.Printf("Failed to create indexes: %v", err)
	}

	return &mongoRepository{coll: coll}, nil
}

func (r *mongoRepository) Search(ctx context.Context, params *model.SearchParams) ([]*model.Listing, int64, error) {
	filter := bson.M{}

	if params.Destination != "" {
		filter["destination"] = bson.M{"$regex": params.Destination, "$options": "i"} // Case-insensitive search
	}
	if params.MinPrice > 0 || params.MaxPrice > 0 {
		priceFilter := bson.M{}
		if params.MinPrice > 0 {
			priceFilter["$gte"] = params.MinPrice
		}
		if params.MaxPrice > 0 {
			priceFilter["$lte"] = params.MaxPrice
		}
		filter["price"] = priceFilter
	}
	if params.Date != "" {
		filter["date"] = params.Date
	}

	count, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "created_at", Value: -1}})
	findOptions.SetSkip(int64((params.Page - 1) * params.Limit))
	findOptions.SetLimit(int64(params.Limit))

	cursor, err := r.coll.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var listings []*model.Listing
	if err := cursor.All(ctx, &listings); err != nil {
		return nil, 0, err
	}

	return listings, count, nil
}

func (r *mongoRepository) Seed(ctx context.Context) error {
	count, _ := r.coll.CountDocuments(ctx, bson.M{})
	if count > 0 {
		return nil // Already seeded
	}

	listings := []interface{}{
		model.Listing{Title: "Paris Gateway", Destination: "Paris", Price: 200, Date: "2026-06-01", AvailableSlots: 10, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		model.Listing{Title: "Tokyo Adventure", Destination: "Tokyo", Price: 300, Date: "2026-07-15", AvailableSlots: 5, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		model.Listing{Title: "New York City Break", Destination: "New York", Price: 250, Date: "2026-08-20", AvailableSlots: 8, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		model.Listing{Title: "Bali Retreat", Destination: "Bali", Price: 150, Date: "2026-09-05", AvailableSlots: 12, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		model.Listing{Title: "London Historical Tour", Destination: "London", Price: 220, Date: "2026-06-10", AvailableSlots: 15, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	_, err := r.coll.InsertMany(ctx, listings)
	return err
}

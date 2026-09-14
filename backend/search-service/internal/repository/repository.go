package repository

import (
	"context"
	"errors"
	"log"
	"regexp"
	"time"

	"github.com/gavinadlan/tripnest/backend/search-service/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ListingRepository interface {
	Search(ctx context.Context, params *model.SearchParams) ([]*model.Listing, int64, error)
	List(ctx context.Context) ([]*model.Listing, error)
	GetByID(ctx context.Context, id string) (*model.Listing, error)
	// Upsert dan Delete hanya dipanggil dari consumer event. Sengaja
	// upsert, bukan insert: event yang sama boleh datang dua kali dan
	// hasil akhirnya harus tetap sama.
	Upsert(ctx context.Context, listing *model.Listing) error
	Delete(ctx context.Context, id string) error
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
		// QuoteMeta wajib: tanpa ini input seperti "(a+)+$" jadi regex
		// katastrofik yang menggantung MongoDB (ReDoS).
		filter["destination"] = bson.M{"$regex": regexp.QuoteMeta(params.Destination), "$options": "i"}
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

func (r *mongoRepository) List(ctx context.Context) ([]*model.Listing, error) {
	cursor, err := r.coll.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var listings []*model.Listing
	if err := cursor.All(ctx, &listings); err != nil {
		return nil, err
	}
	return listings, nil
}

// GetByID melayani halaman detail trip di frontend. Ini read model:
// booking-service tidak lagi membaca dari sini, melainkan dari
// catalog-service yang memiliki datanya.
func (r *mongoRepository) GetByID(ctx context.Context, id string) (*model.Listing, error) {
	var listing model.Listing
	if err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&listing); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &listing, nil
}

// Upsert menulis snapshot trip apa adanya. Read model tidak menghitung
// apa pun sendiri -- angkanya milik catalog-service.
func (r *mongoRepository) Upsert(ctx context.Context, listing *model.Listing) error {
	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"title":           listing.Title,
			"destination":     listing.Destination,
			"price":           listing.Price,
			"date":            listing.Date,
			"total_slots":     listing.TotalSlots,
			"available_slots": listing.AvailableSlots,
			"updated_at":      listing.UpdatedAt,
		},
		"$setOnInsert": bson.M{"created_at": now},
	}
	_, err := r.coll.UpdateByID(ctx, listing.ID, update, options.Update().SetUpsert(true))
	return err
}

// Delete tidak menganggap dokumen yang sudah tidak ada sebagai kesalahan:
// trip.deleted yang terkirim ulang harus tetap berhasil.
func (r *mongoRepository) Delete(ctx context.Context, id string) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

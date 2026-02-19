package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Listing struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Title          string             `json:"title" bson:"title"`
	Destination    string             `json:"destination" bson:"destination"`
	Price          float64            `json:"price" bson:"price"`
	Date           string             `json:"date" bson:"date"` // Using simplified YYYY-MM-DD
	AvailableSlots int                `json:"available_slots" bson:"available_slots"`
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" bson:"updated_at"`
}

type SearchParams struct {
	Destination string
	MinPrice    float64
	MaxPrice    float64
	Date        string
	Page        int
	Limit       int
}

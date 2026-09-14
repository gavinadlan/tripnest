package model

import "time"

// Listing adalah proyeksi read-only dari trip yang dimiliki
// catalog-service. Isinya hanya boleh berubah lewat event trip.upserted /
// trip.deleted -- service ini tidak punya endpoint tulis lagi, karena
// dulu katalog bisa ditulis di dua tempat dan kapasitasnya tidak pernah
// sama.
type Listing struct {
	// ID memakai id trip dari Postgres, bukan ObjectID Mongo. Dengan
	// begitu id yang dilihat frontend, dipakai booking, dan disimpan di
	// read model adalah id yang sama persis.
	ID             string    `json:"id" bson:"_id"`
	Title          string    `json:"title" bson:"title"`
	Destination    string    `json:"destination" bson:"destination"`
	Price          float64   `json:"price" bson:"price"`
	Date           string    `json:"date" bson:"date"` // YYYY-MM-DD
	TotalSlots     int       `json:"total_slots" bson:"total_slots"`
	AvailableSlots int       `json:"available_slots" bson:"available_slots"`
	CreatedAt      time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" bson:"updated_at"`
}

type SearchParams struct {
	Destination string
	MinPrice    float64
	MaxPrice    float64
	Date        string
	Page        int
	Limit       int
}

// TripUpsertedEvent dan TripDeletedEvent harus sebentuk dengan yang
// diterbitkan catalog-service.
type TripUpsertedEvent struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Destination    string  `json:"destination"`
	Price          float64 `json:"price"`
	Date           string  `json:"date"`
	TotalSlots     int     `json:"total_slots"`
	AvailableSlots int     `json:"available_slots"`
	UpdatedAt      string  `json:"updated_at"`
}

type TripDeletedEvent struct {
	ID string `json:"id"`
}

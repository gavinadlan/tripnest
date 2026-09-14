package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/model"
	"github.com/jackc/pgx/v5"
)

var (
	// ErrTripNotFound dibedakan dari error database supaya handler bisa
	// membalas 404, bukan 500.
	ErrTripNotFound = errors.New("trip tidak ditemukan")
	// ErrTripHasReservations mencegah trip yang masih dipesan orang lenyap
	// begitu saja. Kalau trip dihapus, booking yang menunjuk ke sana jadi
	// menggantung dan tidak ada cara membereskannya.
	ErrTripHasReservations = errors.New("trip masih punya reservasi aktif")
)

const activeReservationStatuses = `('RESERVED', 'CONFIRMED')`

// CreateTrip menulis trip, kapasitasnya, dan event outbox-nya dalam SATU
// transaksi.
//
// Inilah inti perbaikannya: sebelumnya trip dibuat di MongoDB oleh
// search-service dan baris inventory-nya tidak pernah dibuat sama sekali,
// sehingga setiap reservasi mengenai nol baris dan selalu gagal.
func (r *postgresRepository) CreateTrip(ctx context.Context, req model.TripRequest) (*model.Trip, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var trip model.Trip
	var tripDate time.Time
	if err := tx.QueryRow(ctx, `
		INSERT INTO trips (title, destination, price, trip_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, title, destination, price, trip_date, created_at, updated_at
	`, req.Title, req.Destination, req.Price, req.Date).Scan(
		&trip.ID, &trip.Title, &trip.Destination, &trip.Price, &tripDate, &trip.CreatedAt, &trip.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("gagal menyimpan trip: %w", err)
	}
	trip.Date = tripDate.Format(model.DateLayout)

	if _, err := tx.Exec(ctx, `
		INSERT INTO inventory (resource_id, total_slots, available_slots, created_at, updated_at)
		VALUES ($1, $2, $2, NOW(), NOW())
	`, trip.ID, req.TotalSlots); err != nil {
		return nil, fmt.Errorf("gagal menyimpan kapasitas trip: %w", err)
	}
	trip.TotalSlots = req.TotalSlots
	trip.AvailableSlots = req.TotalSlots

	if err := enqueueTripUpserted(ctx, tx, trip.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &trip, nil
}

// UpdateTrip mengubah detail trip dan kapasitasnya sekaligus.
//
// available_slots dihitung ulang dari reservasi yang masih aktif, bukan
// disalin mentah dari total: menurunkan kapasitas trip yang sudah terisi
// tidak boleh membuat sisa kursi jadi negatif atau menghapus pesanan
// yang sudah ada.
func (r *postgresRepository) UpdateTrip(ctx context.Context, id string, req model.TripRequest) (*model.Trip, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE trips
		SET title = $2, destination = $3, price = $4, trip_date = $5, updated_at = NOW()
		WHERE id = $1
	`, id, req.Title, req.Destination, req.Price, req.Date)
	if err != nil {
		return nil, fmt.Errorf("gagal memperbarui trip: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrTripNotFound
	}

	reserved, err := lockAndCountReserved(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	available := req.TotalSlots - reserved
	if available < 0 {
		available = 0
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO inventory (resource_id, total_slots, available_slots, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (resource_id) DO UPDATE SET
			total_slots = EXCLUDED.total_slots,
			available_slots = EXCLUDED.available_slots,
			updated_at = NOW()
	`, id, req.TotalSlots, available); err != nil {
		return nil, fmt.Errorf("gagal memperbarui kapasitas trip: %w", err)
	}

	if err := enqueueTripUpserted(ctx, tx, id); err != nil {
		return nil, err
	}

	trip, err := readTrip(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return trip, nil
}

// DeleteTrip menolak menghapus trip yang masih punya reservasi aktif.
// Baris inventory ikut terhapus lewat ON DELETE CASCADE.
func (r *postgresRepository) DeleteTrip(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	reserved, err := lockAndCountReserved(ctx, tx, id)
	if err != nil {
		return err
	}
	if reserved > 0 {
		return ErrTripHasReservations
	}

	// Reservasi yang sudah dilepas tidak menghalangi penghapusan, tapi
	// foreign key-nya tetap harus dibereskan lebih dulu.
	if _, err := tx.Exec(ctx, `DELETE FROM inventory_reservations WHERE resource_id = $1`, id); err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM trips WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrTripNotFound
	}

	if err := insertOutbox(ctx, tx, "trip.deleted", id, model.TripDeletedEvent{ID: id}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *postgresRepository) GetTrip(ctx context.Context, id string) (*model.Trip, error) {
	trip, err := readTrip(ctx, r.db, id)
	if errors.Is(err, ErrTripNotFound) {
		return nil, nil
	}
	return trip, err
}

func (r *postgresRepository) ListTrips(ctx context.Context) ([]model.Trip, error) {
	rows, err := r.db.Query(ctx, tripSelect+` ORDER BY t.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	trips := make([]model.Trip, 0)
	for rows.Next() {
		trip, err := scanTrip(rows)
		if err != nil {
			return nil, err
		}
		trips = append(trips, *trip)
	}
	return trips, rows.Err()
}

// SeedTrips mengisi katalog contoh, dan tidak melakukan apa-apa bila
// katalog sudah berisi. Harganya dalam rupiah dan tanggalnya di masa
// depan -- seed lama memakai angka skala dolar yang lalu ditampilkan
// sebagai "Rp 150".
func (r *postgresRepository) SeedTrips(ctx context.Context) (int, error) {
	var existing int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM trips`).Scan(&existing); err != nil {
		return 0, err
	}
	if existing > 0 {
		return 0, nil
	}

	base := time.Now().UTC().AddDate(0, 1, 0)
	samples := []model.TripRequest{
		{Title: "Bali Retreat 4 Hari", Destination: "Bali", Price: 4500000, Date: base.Format(model.DateLayout), TotalSlots: 12},
		{Title: "Tokyo Adventure", Destination: "Tokyo", Price: 18500000, Date: base.AddDate(0, 0, 14).Format(model.DateLayout), TotalSlots: 5},
		{Title: "Labuan Bajo Sailing", Destination: "Labuan Bajo", Price: 7250000, Date: base.AddDate(0, 0, 21).Format(model.DateLayout), TotalSlots: 8},
		{Title: "Paris Getaway", Destination: "Paris", Price: 24000000, Date: base.AddDate(0, 1, 0).Format(model.DateLayout), TotalSlots: 6},
		{Title: "Raja Ampat Explorer", Destination: "Raja Ampat", Price: 12750000, Date: base.AddDate(0, 1, 10).Format(model.DateLayout), TotalSlots: 10},
	}

	for _, sample := range samples {
		if _, err := r.CreateTrip(ctx, sample); err != nil {
			return 0, err
		}
	}
	return len(samples), nil
}

const tripSelect = `
	SELECT
		t.id,
		t.title,
		t.destination,
		t.price,
		t.trip_date,
		COALESCE(i.total_slots, 0),
		COALESCE(i.available_slots, 0),
		t.created_at,
		t.updated_at
	FROM trips t
	LEFT JOIN inventory i ON i.resource_id = t.id`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTrip(row rowScanner) (*model.Trip, error) {
	var trip model.Trip
	var tripDate time.Time
	if err := row.Scan(
		&trip.ID, &trip.Title, &trip.Destination, &trip.Price, &tripDate,
		&trip.TotalSlots, &trip.AvailableSlots, &trip.CreatedAt, &trip.UpdatedAt,
	); err != nil {
		return nil, err
	}
	trip.Date = tripDate.Format(model.DateLayout)
	return &trip, nil
}

// rowQuerier dipenuhi baik oleh pool maupun oleh transaksi, sehingga
// helper di bawah bisa dipakai di dalam dan di luar transaksi tanpa
// menduplikasi query-nya.
type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func readTrip(ctx context.Context, q rowQuerier, id string) (*model.Trip, error) {
	trip, err := scanTrip(q.QueryRow(ctx, tripSelect+` WHERE t.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTripNotFound
	}
	if err != nil {
		return nil, err
	}
	return trip, nil
}

// lockAndCountReserved mengunci baris inventory lebih dulu, lalu
// menjumlahkan kursi yang sedang dipegang reservasi aktif.
//
// Kuncinya penting: ReserveSlot mengunci baris yang sama saat mengurangi
// stok, jadi keduanya berjalan bergantian, bukan bersamaan di atas
// hitungan yang sudah basi.
func lockAndCountReserved(ctx context.Context, tx pgx.Tx, resourceID string) (int, error) {
	var locked string
	err := tx.QueryRow(ctx, `SELECT resource_id FROM inventory WHERE resource_id = $1 FOR UPDATE`, resourceID).Scan(&locked)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}

	var reserved int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(quantity), 0)
		FROM inventory_reservations
		WHERE resource_id = $1 AND status IN `+activeReservationStatuses,
		resourceID,
	).Scan(&reserved); err != nil {
		return 0, err
	}
	return reserved, nil
}

// enqueueTripUpserted menulis snapshot trip terbaru ke outbox, di dalam
// transaksi yang mengubahnya. Read model karena itu tidak pernah bisa
// menerima event untuk perubahan yang ternyata di-rollback.
func enqueueTripUpserted(ctx context.Context, tx pgx.Tx, tripID string) error {
	trip, err := readTrip(ctx, tx, tripID)
	if err != nil {
		return err
	}
	event := model.TripUpsertedEvent{
		ID:             trip.ID,
		Title:          trip.Title,
		Destination:    trip.Destination,
		Price:          trip.Price,
		Date:           trip.Date,
		TotalSlots:     trip.TotalSlots,
		AvailableSlots: trip.AvailableSlots,
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339Nano),
	}
	return insertOutbox(ctx, tx, "trip.upserted", trip.ID, event)
}

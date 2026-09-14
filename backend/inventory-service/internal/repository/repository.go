package repository

import (
	"context"
	"errors"

	"github.com/gavinadlan/tripnest/backend/common/outbox"
	"github.com/gavinadlan/tripnest/backend/common/pgtx"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReserveOutcome membedakan tiga hasil yang perlakuannya berbeda.
// Sebelumnya "kursi habis" dikembalikan sebagai error biasa dan berakhir
// di log.Printf, sehingga booking tetap lanjut ke pembayaran.
type ReserveOutcome int

const (
	ReserveOutcomeReserved ReserveOutcome = iota
	// ReserveOutcomeDuplicate berarti reservasi untuk booking ini sudah
	// ada. Event Kafka bisa datang dua kali; ini bukan kesalahan.
	ReserveOutcomeDuplicate
	// ReserveOutcomeRejected berarti kursi tidak cukup atau trip-nya sudah
	// tidak ada. Pemanggil harus membatalkan booking-nya.
	ReserveOutcomeRejected
)

type InventoryRepository interface {
	// ReserveSlot menerima event booking utuh, bukan hanya id dan jumlah:
	// isinya disalin ke inventory.reserved supaya payment-service punya
	// semua yang dibutuhkan tanpa memanggil balik booking-service.
	ReserveSlot(ctx context.Context, event model.BookingCreatedEvent) (ReserveOutcome, error)
	ConfirmReservation(ctx context.Context, bookingID string) (bool, error)
	ReleaseReservation(ctx context.Context, bookingID string, allowConfirmed bool) (bool, error)
	UpdateInventory(ctx context.Context, resourceID string, totalSlots int) error
	GetInventoryByResourceID(ctx context.Context, resourceID string) (*model.Inventory, error)
	ListInventory(ctx context.Context) ([]model.Inventory, error)

	CreateTrip(ctx context.Context, req model.TripRequest) (*model.Trip, error)
	UpdateTrip(ctx context.Context, id string, req model.TripRequest) (*model.Trip, error)
	DeleteTrip(ctx context.Context, id string) error
	GetTrip(ctx context.Context, id string) (*model.Trip, error)
	ListTrips(ctx context.Context) ([]model.Trip, error)
	SeedTrips(ctx context.Context) (int, error)

	// Transact menjalankan fn dalam satu transaksi. Semua method di atas
	// yang dipanggil dengan ctx milik fn ikut di dalamnya.
	Transact(ctx context.Context, fn func(ctx context.Context) error) error
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) InventoryRepository {
	return &postgresRepository{db: pool}
}

func (r *postgresRepository) Transact(ctx context.Context, fn func(ctx context.Context) error) error {
	return pgtx.Transact(ctx, r.db, fn)
}

// q mengembalikan transaksi yang dibawa ctx, atau pool bila tidak ada.
func (r *postgresRepository) q(ctx context.Context) pgtx.Querier {
	return pgtx.From(ctx, r.db)
}

// ReserveSlot menahan kursi untuk sebuah booking.
//
// Pengurangan stok, pencatatan reservasi, dan event yang menyusul semuanya
// berada dalam satu transaksi. Kalau kursi tidak cukup, transaksi yang
// sama menulis inventory.rejected ke outbox -- booking-service memakainya
// untuk membatalkan booking, sehingga tidak ada lagi booking yang lanjut
// ke pembayaran padahal kursinya sudah habis.
func (r *postgresRepository) ReserveSlot(ctx context.Context, event model.BookingCreatedEvent) (ReserveOutcome, error) {
	outcome := ReserveOutcomeRejected
	err := r.Transact(ctx, func(ctx context.Context) error {
		var err error
		outcome, err = r.reserveSlot(ctx, event)
		return err
	})
	return outcome, err
}

// reserveSlot adalah isi ReserveSlot; berjalan di dalam transaksi yang
// dibawa ctx. Kalau consumer sudah membuka transaksi (untuk menulis
// processed_messages bersama efeknya), kita ikut di sana.
func (r *postgresRepository) reserveSlot(ctx context.Context, event model.BookingCreatedEvent) (ReserveOutcome, error) {
	bookingID := event.BookingID
	resourceID := event.ResourceID
	quantity := event.Seats()
	tx := r.q(ctx)

	var existingStatus string
	err := tx.QueryRow(ctx, `SELECT status FROM inventory_reservations WHERE booking_id = $1`, bookingID).Scan(&existingStatus)
	if err == nil {
		return ReserveOutcomeDuplicate, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ReserveOutcomeRejected, err
	}

	updateTag, err := tx.Exec(ctx, `
		UPDATE inventory
		SET available_slots = available_slots - $2, updated_at = NOW()
		WHERE resource_id = $1 AND available_slots >= $2
	`, resourceID, quantity)
	if err != nil {
		return ReserveOutcomeRejected, err
	}

	if updateTag.RowsAffected() == 0 {
		reason := "kursi tidak mencukupi"
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM inventory WHERE resource_id = $1)`, resourceID).Scan(&exists); err != nil {
			return ReserveOutcomeRejected, err
		}
		if !exists {
			reason = "trip tidak ditemukan"
		}
		rejected := model.InventoryRejectedEvent{
			BookingID:  bookingID,
			ResourceID: resourceID,
			Quantity:   quantity,
			Reason:     reason,
		}
		if err := outbox.Enqueue(ctx, tx, "inventory.rejected", bookingID, rejected); err != nil {
			return ReserveOutcomeRejected, err
		}
		return ReserveOutcomeRejected, nil
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO inventory_reservations (booking_id, resource_id, quantity, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, bookingID, resourceID, quantity, model.ReservationStatusReserved); err != nil {
		return ReserveOutcomeRejected, err
	}

	// Sisa kursi berubah, jadi read model harus tahu. Tanpa ini angka
	// "sisa N kursi" di katalog tidak pernah berkurang.
	if err := enqueueTripUpserted(ctx, tx, resourceID); err != nil {
		return ReserveOutcomeRejected, err
	}

	// Izin menagih terbit dalam transaksi yang sama dengan pengurangan
	// kursinya. Tidak ada celah di mana kursi sudah berkurang tapi tidak
	// ada yang tahu, atau sebaliknya.
	reserved := model.InventoryReservedEvent{
		BookingID:   bookingID,
		UserID:      event.UserID,
		ResourceID:  resourceID,
		Quantity:    quantity,
		TotalAmount: event.TotalAmount,
		ExpiresAt:   event.ExpiresAt,
	}
	if err := outbox.Enqueue(ctx, tx, "inventory.reserved", bookingID, reserved); err != nil {
		return ReserveOutcomeRejected, err
	}

	return ReserveOutcomeReserved, nil
}

// ConfirmReservation tidak mengubah sisa kursi: kursinya sudah ditahan
// sejak RESERVED. Yang berubah hanya sifat tahanannya, dari sementara
// jadi tetap.
func (r *postgresRepository) ConfirmReservation(ctx context.Context, bookingID string) (bool, error) {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE inventory_reservations
		SET status = $1, updated_at = NOW()
		WHERE booking_id = $2 AND status = $3
	`, model.ReservationStatusConfirmed, bookingID, model.ReservationStatusReserved)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// ReleaseReservation mengembalikan kursi ke stok.
//
// allowConfirmed dipakai untuk pembatalan eksplisit (dan nanti refund).
// Sebelumnya reservasi CONFIRMED tidak pernah bisa dilepas dengan alasan
// apa pun, sehingga kursi yang dibatalkan hilang selamanya.
func (r *postgresRepository) ReleaseReservation(ctx context.Context, bookingID string, allowConfirmed bool) (bool, error) {
	released := false
	err := r.Transact(ctx, func(ctx context.Context) error {
		var err error
		released, err = r.releaseReservation(ctx, bookingID, allowConfirmed)
		return err
	})
	return released, err
}

func (r *postgresRepository) releaseReservation(ctx context.Context, bookingID string, allowConfirmed bool) (bool, error) {
	tx := r.q(ctx)

	var resourceID, status string
	var quantity int
	err := tx.QueryRow(ctx, `
		SELECT resource_id, status, quantity
		FROM inventory_reservations
		WHERE booking_id = $1
		FOR UPDATE
	`, bookingID).Scan(&resourceID, &status, &quantity)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if status == model.ReservationStatusReleased {
		return false, nil
	}
	if status == model.ReservationStatusConfirmed && !allowConfirmed {
		return false, nil
	}

	if _, err := tx.Exec(ctx, `
		UPDATE inventory_reservations
		SET status = $1, updated_at = NOW()
		WHERE booking_id = $2
	`, model.ReservationStatusReleased, bookingID); err != nil {
		return false, err
	}

	// LEAST menjaga sisa kursi tidak pernah melewati total, misalnya bila
	// admin menurunkan kapasitas selagi ada reservasi berjalan.
	if _, err := tx.Exec(ctx, `
		UPDATE inventory
		SET available_slots = LEAST(total_slots, available_slots + $2), updated_at = NOW()
		WHERE resource_id = $1
	`, resourceID, quantity); err != nil {
		return false, err
	}

	if err := enqueueTripUpserted(ctx, tx, resourceID); err != nil {
		return false, err
	}

	return true, nil
}

// UpdateInventory menyesuaikan kapasitas tanpa menyentuh detail trip.
// Pembuatan kapasitas baru tidak lagi lewat sini: kapasitas selalu lahir
// bersama trip-nya di CreateTrip.
func (r *postgresRepository) UpdateInventory(ctx context.Context, resourceID string, totalSlots int) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM trips WHERE id = $1)`, resourceID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrTripNotFound
	}

	reserved, err := lockAndCountReserved(ctx, tx, resourceID)
	if err != nil {
		return err
	}
	available := totalSlots - reserved
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
	`, resourceID, totalSlots, available); err != nil {
		return err
	}

	if err := enqueueTripUpserted(ctx, tx, resourceID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *postgresRepository) GetInventoryByResourceID(ctx context.Context, resourceID string) (*model.Inventory, error) {
	var inv model.Inventory
	err := r.q(ctx).QueryRow(ctx, `
		SELECT
			i.resource_id,
			i.total_slots,
			i.available_slots,
			(i.total_slots - i.available_slots) AS reserved_slots,
			i.created_at,
			i.updated_at
		FROM inventory i
		WHERE i.resource_id = $1
	`, resourceID).Scan(&inv.ResourceID, &inv.TotalSlots, &inv.AvailableSlots, &inv.ReservedSlots, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &inv, nil
}

func (r *postgresRepository) ListInventory(ctx context.Context) ([]model.Inventory, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT
			resource_id,
			total_slots,
			available_slots,
			(total_slots - available_slots) AS reserved_slots,
			created_at,
			updated_at
		FROM inventory
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Inventory, 0)
	for rows.Next() {
		var inv model.Inventory
		if err := rows.Scan(&inv.ResourceID, &inv.TotalSlots, &inv.AvailableSlots, &inv.ReservedSlots, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, inv)
	}
	return items, rows.Err()
}

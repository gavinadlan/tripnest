package repository

import (
	"context"
	"encoding/json"
	"errors"

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

	FetchPendingOutboxEvents(ctx context.Context, limit int) ([]OutboxEvent, error)
	MarkOutboxEventPublished(ctx context.Context, id string) error
	MarkOutboxEventFailed(ctx context.Context, id string, errMsg string) error
	MarkMessageProcessed(ctx context.Context, consumerGroup, topic, messageKey string) (bool, error)
	Close()
}

type postgresRepository struct {
	db *pgxpool.Pool
}

type OutboxEvent struct {
	ID      string
	Topic   string
	Key     string
	Payload json.RawMessage
}

func NewPostgresRepository(connString string) (InventoryRepository, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	return &postgresRepository{db: pool}, nil
}

func (r *postgresRepository) Close() {
	r.db.Close()
}

// ReserveSlot menahan kursi untuk sebuah booking.
//
// Pengurangan stok, pencatatan reservasi, dan event yang menyusul semuanya
// berada dalam satu transaksi. Kalau kursi tidak cukup, transaksi yang
// sama menulis inventory.rejected ke outbox -- booking-service memakainya
// untuk membatalkan booking, sehingga tidak ada lagi booking yang lanjut
// ke pembayaran padahal kursinya sudah habis.
func (r *postgresRepository) ReserveSlot(ctx context.Context, event model.BookingCreatedEvent) (ReserveOutcome, error) {
	bookingID := event.BookingID
	resourceID := event.ResourceID
	quantity := event.Seats()

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ReserveOutcomeRejected, err
	}
	defer tx.Rollback(ctx)

	var existingStatus string
	err = tx.QueryRow(ctx, `SELECT status FROM inventory_reservations WHERE booking_id = $1`, bookingID).Scan(&existingStatus)
	if err == nil {
		return ReserveOutcomeDuplicate, tx.Commit(ctx)
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
		if err := insertOutbox(ctx, tx, "inventory.rejected", bookingID, rejected); err != nil {
			return ReserveOutcomeRejected, err
		}
		return ReserveOutcomeRejected, tx.Commit(ctx)
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
	if err := insertOutbox(ctx, tx, "inventory.reserved", bookingID, reserved); err != nil {
		return ReserveOutcomeRejected, err
	}

	return ReserveOutcomeReserved, tx.Commit(ctx)
}

// ConfirmReservation tidak mengubah sisa kursi: kursinya sudah ditahan
// sejak RESERVED. Yang berubah hanya sifat tahanannya, dari sementara
// jadi tetap.
func (r *postgresRepository) ConfirmReservation(ctx context.Context, bookingID string) (bool, error) {
	tag, err := r.db.Exec(ctx, `
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
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var resourceID, status string
	var quantity int
	err = tx.QueryRow(ctx, `
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
		return false, tx.Commit(ctx)
	}
	if status == model.ReservationStatusConfirmed && !allowConfirmed {
		return false, tx.Commit(ctx)
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

	return true, tx.Commit(ctx)
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
	err := r.db.QueryRow(ctx, `
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
	rows, err := r.db.Query(ctx, `
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

// insertOutbox selalu dipanggil dengan transaksi yang juga membawa
// perubahan datanya. Event dan efeknya lolos atau gagal bersama-sama.
func insertOutbox(ctx context.Context, tx pgx.Tx, topic, key string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events (topic, message_key, payload, status, retry_count, created_at, updated_at)
		VALUES ($1, $2, $3, 'PENDING', 0, NOW(), NOW())
	`, topic, key, data)
	return err
}

func (r *postgresRepository) FetchPendingOutboxEvents(ctx context.Context, limit int) ([]OutboxEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, topic, message_key, payload
		FROM outbox_events
		WHERE status = 'PENDING'
		ORDER BY created_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]OutboxEvent, 0)
	for rows.Next() {
		var evt OutboxEvent
		if err := rows.Scan(&evt.ID, &evt.Topic, &evt.Key, &evt.Payload); err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, rows.Err()
}

func (r *postgresRepository) MarkOutboxEventPublished(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE outbox_events
		SET status = 'PUBLISHED', published_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, id)
	return err
}

func (r *postgresRepository) MarkOutboxEventFailed(ctx context.Context, id string, errMsg string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE outbox_events
		SET retry_count = retry_count + 1, last_error = $2, updated_at = NOW()
		WHERE id = $1
	`, id, errMsg)
	return err
}

func (r *postgresRepository) MarkMessageProcessed(ctx context.Context, consumerGroup, topic, messageKey string) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		INSERT INTO processed_messages (consumer_group, topic, message_key, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (consumer_group, topic, message_key) DO NOTHING
	`, consumerGroup, topic, messageKey)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

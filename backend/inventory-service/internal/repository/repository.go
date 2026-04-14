package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InventoryRepository interface {
	ReserveSlot(ctx context.Context, bookingID, resourceID string) (bool, error)
	ConfirmReservation(ctx context.Context, bookingID string) (bool, error)
	ReleaseReservation(ctx context.Context, bookingID string) (bool, error)
	UpsertInventory(ctx context.Context, req model.UpsertInventoryRequest) error
	UpdateInventory(ctx context.Context, resourceID string, totalSlots int) error
	GetInventoryByResourceID(ctx context.Context, resourceID string) (*model.Inventory, error)
	ListInventory(ctx context.Context) ([]model.Inventory, error)
	AddOutboxEvent(ctx context.Context, topic, key string, payload interface{}) error
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

func (r *postgresRepository) ReserveSlot(ctx context.Context, bookingID, resourceID string) (bool, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var existingStatus string
	err = tx.QueryRow(ctx, `SELECT status FROM inventory_reservations WHERE booking_id = $1`, bookingID).Scan(&existingStatus)
	if err == nil {
		return true, tx.Commit(ctx)
	}
	if err != nil && err != pgx.ErrNoRows {
		return false, err
	}

	updateTag, err := tx.Exec(ctx, `
		UPDATE inventory
		SET available_slots = available_slots - 1, updated_at = NOW()
		WHERE resource_id = $1 AND available_slots > 0
	`, resourceID)
	if err != nil {
		return false, err
	}
	if updateTag.RowsAffected() == 0 {
		return false, fmt.Errorf("inventory unavailable for resource %s", resourceID)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO inventory_reservations (booking_id, resource_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, bookingID, resourceID, model.ReservationStatusReserved); err != nil {
		return false, err
	}

	return true, tx.Commit(ctx)
}

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

func (r *postgresRepository) ReleaseReservation(ctx context.Context, bookingID string) (bool, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var resourceID string
	var status string
	err = tx.QueryRow(ctx, `
		SELECT resource_id, status
		FROM inventory_reservations
		WHERE booking_id = $1
	`, bookingID).Scan(&resourceID, &status)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if status == model.ReservationStatusReleased {
		return false, tx.Commit(ctx)
	}
	if status == model.ReservationStatusConfirmed {
		return false, tx.Commit(ctx)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE inventory_reservations
		SET status = $1, updated_at = NOW()
		WHERE booking_id = $2
	`, model.ReservationStatusReleased, bookingID); err != nil {
		return false, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE inventory
		SET available_slots = available_slots + 1, updated_at = NOW()
		WHERE resource_id = $1
	`, resourceID); err != nil {
		return false, err
	}

	return true, tx.Commit(ctx)
}

func (r *postgresRepository) UpsertInventory(ctx context.Context, req model.UpsertInventoryRequest) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO inventory (resource_id, total_slots, available_slots, created_at, updated_at)
		VALUES ($1, $2, $2, NOW(), NOW())
		ON CONFLICT (resource_id) DO UPDATE SET
			total_slots = EXCLUDED.total_slots,
			available_slots = GREATEST(0, EXCLUDED.total_slots - (
				SELECT COUNT(*)
				FROM inventory_reservations
				WHERE inventory_reservations.resource_id = EXCLUDED.resource_id
				AND inventory_reservations.status IN ('RESERVED', 'CONFIRMED')
			)),
			updated_at = NOW()
	`, req.ResourceID, req.TotalSlots)
	return err
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
		if err == pgx.ErrNoRows {
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

func (r *postgresRepository) UpdateInventory(ctx context.Context, resourceID string, totalSlots int) error {
	return r.UpsertInventory(ctx, model.UpsertInventoryRequest{
		ResourceID: resourceID,
		TotalSlots: totalSlots,
	})
}

func (r *postgresRepository) AddOutboxEvent(ctx context.Context, topic, key string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
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

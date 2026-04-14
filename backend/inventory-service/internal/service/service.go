package service

import (
	"context"
	"log"
	"time"

	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/msgbroker"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/repository"
)

type InventoryService interface {
	ReserveSlot(ctx context.Context, event model.BookingCreatedEvent) error
	ConfirmReservation(ctx context.Context, bookingID string) error
	ReleaseReservation(ctx context.Context, bookingID string) error
	UpsertInventory(ctx context.Context, req model.UpsertInventoryRequest) error
	UpdateInventory(ctx context.Context, resourceID string, totalSlots int) error
	GetInventoryByResourceID(ctx context.Context, resourceID string) (*model.Inventory, error)
	ListInventory(ctx context.Context) ([]model.Inventory, error)
}

type inventoryService struct {
	repo repository.InventoryRepository
}

func NewInventoryService(repo repository.InventoryRepository, _ *msgbroker.Producer) InventoryService {
	return &inventoryService{
		repo: repo,
	}
}

func (s *inventoryService) ReserveSlot(ctx context.Context, event model.BookingCreatedEvent) error {
	ok, err := s.repo.ReserveSlot(ctx, event.BookingID, event.ResourceID)
	if err != nil {
		return err
	}
	if !ok {
		log.Printf("Reservation no-op for booking %s", event.BookingID)
		return nil
	}
	log.Printf("Reserved slot booking_id=%s resource_id=%s", event.BookingID, event.ResourceID)

	publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	statusEvent := model.InventoryStatusEvent{
		BookingID:  event.BookingID,
		ResourceID: event.ResourceID,
		Status:     model.ReservationStatusReserved,
	}
	if err := s.repo.AddOutboxEvent(publishCtx, "inventory.reserved", event.BookingID, statusEvent); err != nil {
		log.Printf("failed to enqueue inventory.reserved for booking %s: %v", event.BookingID, err)
	}
	return nil
}

func (s *inventoryService) ConfirmReservation(ctx context.Context, bookingID string) error {
	ok, err := s.repo.ConfirmReservation(ctx, bookingID)
	if err != nil {
		return err
	}
	if !ok {
		log.Printf("Confirm reservation no-op for booking %s", bookingID)
		return nil
	}
	log.Printf("Confirmed reservation booking_id=%s", bookingID)
	return nil
}

func (s *inventoryService) ReleaseReservation(ctx context.Context, bookingID string) error {
	ok, err := s.repo.ReleaseReservation(ctx, bookingID)
	if err != nil {
		return err
	}
	if !ok {
		log.Printf("Release reservation no-op for booking %s", bookingID)
		return nil
	}
	log.Printf("Released reservation booking_id=%s", bookingID)
	return nil
}

func (s *inventoryService) UpsertInventory(ctx context.Context, req model.UpsertInventoryRequest) error {
	return s.repo.UpsertInventory(ctx, req)
}

func (s *inventoryService) GetInventoryByResourceID(ctx context.Context, resourceID string) (*model.Inventory, error) {
	return s.repo.GetInventoryByResourceID(ctx, resourceID)
}

func (s *inventoryService) ListInventory(ctx context.Context) ([]model.Inventory, error) {
	return s.repo.ListInventory(ctx)
}

func (s *inventoryService) UpdateInventory(ctx context.Context, resourceID string, totalSlots int) error {
	return s.repo.UpdateInventory(ctx, resourceID, totalSlots)
}

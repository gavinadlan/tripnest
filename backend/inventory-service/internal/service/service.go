package service

import (
	"context"
	"log"

	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/repository"
)

// CatalogService memiliki trip beserta kapasitasnya. Nama lamanya
// "inventory" hanya menyebut separuh tugasnya sejak trip pindah ke sini.
type CatalogService interface {
	ReserveSlot(ctx context.Context, event model.BookingCreatedEvent) error
	ConfirmReservation(ctx context.Context, bookingID string) error
	ReleaseReservation(ctx context.Context, bookingID string) error
	// ReleaseCancelled dipakai untuk pembatalan eksplisit: reservasi yang
	// sudah CONFIRMED pun dilepas, karena bookingnya memang dibatalkan.
	ReleaseCancelled(ctx context.Context, bookingID string) error
	UpdateInventory(ctx context.Context, resourceID string, totalSlots int) error
	GetInventoryByResourceID(ctx context.Context, resourceID string) (*model.Inventory, error)
	ListInventory(ctx context.Context) ([]model.Inventory, error)

	CreateTrip(ctx context.Context, req model.TripRequest) (*model.Trip, error)
	UpdateTrip(ctx context.Context, id string, req model.TripRequest) (*model.Trip, error)
	DeleteTrip(ctx context.Context, id string) error
	GetTrip(ctx context.Context, id string) (*model.Trip, error)
	ListTrips(ctx context.Context) ([]model.Trip, error)
	SeedTrips(ctx context.Context) (int, error)
}

type catalogService struct {
	repo repository.InventoryRepository
}

func NewCatalogService(repo repository.InventoryRepository) CatalogService {
	return &catalogService{repo: repo}
}

func (s *catalogService) ReserveSlot(ctx context.Context, event model.BookingCreatedEvent) error {
	outcome, err := s.repo.ReserveSlot(ctx, event)
	if err != nil {
		return err
	}

	switch outcome {
	case repository.ReserveOutcomeReserved:
		log.Printf("Kursi ditahan, inventory.reserved diterbitkan booking_id=%s resource_id=%s qty=%d", event.BookingID, event.ResourceID, event.Seats())
	case repository.ReserveOutcomeDuplicate:
		log.Printf("Reservasi sudah ada, event diabaikan booking_id=%s", event.BookingID)
	case repository.ReserveOutcomeRejected:
		// Penolakannya sudah ditulis ke outbox dalam transaksi yang sama,
		// jadi booking-service pasti mendengarnya. Ini bukan error.
		log.Printf("Kursi tidak tersedia, inventory.rejected diterbitkan booking_id=%s resource_id=%s", event.BookingID, event.ResourceID)
	}
	return nil
}

func (s *catalogService) ConfirmReservation(ctx context.Context, bookingID string) error {
	ok, err := s.repo.ConfirmReservation(ctx, bookingID)
	if err != nil {
		return err
	}
	if !ok {
		log.Printf("Konfirmasi reservasi tidak mengubah apa pun booking_id=%s", bookingID)
		return nil
	}
	log.Printf("Reservasi dikonfirmasi booking_id=%s", bookingID)
	return nil
}

func (s *catalogService) ReleaseReservation(ctx context.Context, bookingID string) error {
	return s.release(ctx, bookingID, false)
}

func (s *catalogService) ReleaseCancelled(ctx context.Context, bookingID string) error {
	return s.release(ctx, bookingID, true)
}

func (s *catalogService) release(ctx context.Context, bookingID string, allowConfirmed bool) error {
	ok, err := s.repo.ReleaseReservation(ctx, bookingID, allowConfirmed)
	if err != nil {
		return err
	}
	if !ok {
		log.Printf("Pelepasan reservasi tidak mengubah apa pun booking_id=%s", bookingID)
		return nil
	}
	log.Printf("Reservasi dilepas booking_id=%s allow_confirmed=%t", bookingID, allowConfirmed)
	return nil
}

func (s *catalogService) UpdateInventory(ctx context.Context, resourceID string, totalSlots int) error {
	return s.repo.UpdateInventory(ctx, resourceID, totalSlots)
}

func (s *catalogService) GetInventoryByResourceID(ctx context.Context, resourceID string) (*model.Inventory, error) {
	return s.repo.GetInventoryByResourceID(ctx, resourceID)
}

func (s *catalogService) ListInventory(ctx context.Context) ([]model.Inventory, error) {
	return s.repo.ListInventory(ctx)
}

func (s *catalogService) CreateTrip(ctx context.Context, req model.TripRequest) (*model.Trip, error) {
	return s.repo.CreateTrip(ctx, req)
}

func (s *catalogService) UpdateTrip(ctx context.Context, id string, req model.TripRequest) (*model.Trip, error) {
	return s.repo.UpdateTrip(ctx, id, req)
}

func (s *catalogService) DeleteTrip(ctx context.Context, id string) error {
	return s.repo.DeleteTrip(ctx, id)
}

func (s *catalogService) GetTrip(ctx context.Context, id string) (*model.Trip, error) {
	return s.repo.GetTrip(ctx, id)
}

func (s *catalogService) ListTrips(ctx context.Context) ([]model.Trip, error) {
	return s.repo.ListTrips(ctx)
}

func (s *catalogService) SeedTrips(ctx context.Context) (int, error) {
	return s.repo.SeedTrips(ctx)
}

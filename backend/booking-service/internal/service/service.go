package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gavinadlan/tripnest/backend/booking-service/internal/events"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/repository"
)

type BookingService interface {
	CreateBooking(ctx context.Context, req *model.CreateBookingRequest) (*model.Booking, error)
	GetBooking(ctx context.Context, id string) (*model.Booking, error)
	ListBookings(ctx context.Context, status, createdDate string) ([]model.Booking, error)
	ConfirmBooking(ctx context.Context, bookingID string) error
	CancelBooking(ctx context.Context, bookingID string) error
	ExpirePendingBookings(ctx context.Context) error
}

type bookingService struct {
	repo         repository.BookingRepository
	expiryWindow time.Duration
}

func NewBookingService(repo repository.BookingRepository, _ *events.KafkaProducer, bookingExpiryMinutes int) BookingService {
	return &bookingService{
		repo:         repo,
		expiryWindow: time.Duration(bookingExpiryMinutes) * time.Minute,
	}
}

func (s *bookingService) CreateBooking(ctx context.Context, req *model.CreateBookingRequest) (*model.Booking, error) {
	booking := &model.Booking{
		UserID:      req.UserID,
		ResourceID:  req.ResourceID,
		TotalAmount: req.TotalAmount,
		Status:      model.BookingStatusPendingPayment,
		ExpiresAt:   time.Now().UTC().Add(s.expiryWindow),
	}

	if err := s.repo.Create(ctx, booking); err != nil {
		return nil, err
	}

	// Publish event
	event := model.BookingCreatedEvent{
		BookingID:   booking.ID,
		UserID:      booking.UserID,
		ResourceID:  booking.ResourceID,
		TotalAmount: booking.TotalAmount,
		ExpiresAt:   booking.ExpiresAt.Format(time.RFC3339),
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.repo.AddOutboxEvent(ctx, "booking.created", booking.ID, event); err != nil {
			fmt.Printf("Failed to enqueue booking.created outbox event: %v\n", err)
		}
	}()

	return booking, nil
}

func (s *bookingService) GetBooking(ctx context.Context, id string) (*model.Booking, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *bookingService) ListBookings(ctx context.Context, status, createdDate string) ([]model.Booking, error) {
	return s.repo.List(ctx, status, createdDate)
}

func (s *bookingService) ConfirmBooking(ctx context.Context, bookingID string) error {
	log.Printf("Confirming booking %s", bookingID)
	updated, err := s.repo.UpdateStatusIfCurrent(ctx, bookingID, model.BookingStatusPendingPayment, model.BookingStatusConfirmed)
	if err != nil {
		return err
	}
	if !updated {
		return nil
	}
	return nil
}

func (s *bookingService) CancelBooking(ctx context.Context, bookingID string) error {
	log.Printf("Cancelling booking %s", bookingID)
	updated, err := s.repo.UpdateStatusIfCurrent(ctx, bookingID, model.BookingStatusPendingPayment, model.BookingStatusCancelled)
	if err != nil {
		return err
	}
	if !updated {
		return nil
	}
	return nil
}

func (s *bookingService) ExpirePendingBookings(ctx context.Context) error {
	expiredIDs, err := s.repo.ExpirePendingBookings(ctx, time.Now().UTC())
	if err != nil {
		return err
	}
	for _, bookingID := range expiredIDs {
		log.Printf("Expired booking %s", bookingID)
		expiredEvent := model.BookingExpiredEvent{
			BookingID: bookingID,
			Status:    model.BookingStatusExpired,
		}
		publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := s.repo.AddOutboxEvent(publishCtx, "booking.expired", bookingID, expiredEvent); err != nil {
			log.Printf("Failed to enqueue booking.expired for %s: %v", bookingID, err)
		}
		cancel()
	}
	return nil
}

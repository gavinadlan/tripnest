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
	ConfirmBooking(ctx context.Context, bookingID string) error
	CancelBooking(ctx context.Context, bookingID string) error
}

type bookingService struct {
	repo     repository.BookingRepository
	producer *events.KafkaProducer
}

func NewBookingService(repo repository.BookingRepository, producer *events.KafkaProducer) BookingService {
	return &bookingService{repo: repo, producer: producer}
}

func (s *bookingService) CreateBooking(ctx context.Context, req *model.CreateBookingRequest) (*model.Booking, error) {
	booking := &model.Booking{
		UserID:      req.UserID,
		ResourceID:  req.ResourceID,
		TotalAmount: req.TotalAmount,
		Status:      "PENDING",
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
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.producer.Publish(ctx, "booking.created", booking.ID, event); err != nil {
			fmt.Printf("Failed to publish booking.created event: %v\n", err)
		}
	}()

	return booking, nil
}

func (s *bookingService) GetBooking(ctx context.Context, id string) (*model.Booking, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *bookingService) ConfirmBooking(ctx context.Context, bookingID string) error {
	log.Printf("Confirming booking %s", bookingID)
	// Additional logic here: e.g. send confirmation email event
	return s.repo.UpdateStatus(ctx, bookingID, "CONFIRMED")
}

func (s *bookingService) CancelBooking(ctx context.Context, bookingID string) error {
	log.Printf("Cancelling booking %s", bookingID)
	// Additional logic here: e.g. revert resource reservation
	return s.repo.UpdateStatus(ctx, bookingID, "CANCELLED")
}

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/gavinadlan/tripnest/backend/booking-service/internal/events"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/repository"
)

type BookingService interface {
	CreateBooking(ctx context.Context, req *model.CreateBookingRequest) (*model.Booking, error)
	GetBooking(ctx context.Context, id string) (*model.Booking, error)
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

	// Use background context for async publishing or same context?
	// Async is better for latency, but we want reliability.Saga logic might require confirmation.
	// For now, synchronous publish.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.producer.Publish(ctx, "booking.created", booking.ID, event); err != nil {
			fmt.Printf("Failed to publish booking.created event: %v\n", err)
			// TODO: Implement outbox pattern or retry
		}
	}()

	return booking, nil
}

func (s *bookingService) GetBooking(ctx context.Context, id string) (*model.Booking, error) {
	return s.repo.GetByID(ctx, id)
}

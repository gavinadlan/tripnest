package service

import (
	"context"
	"fmt"
	"log"

	"github.com/gavinadlan/tripnest/backend/payment-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/msgbroker"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/repository"
)

type PaymentService interface {
	ProcessPayment(ctx context.Context, bookingEvent model.BookingEvent) error
}

type paymentService struct {
	repo     repository.PaymentRepository
	producer *msgbroker.Producer
}

func NewPaymentService(repo repository.PaymentRepository, producer *msgbroker.Producer) PaymentService {
	return &paymentService{repo: repo, producer: producer}
}

func (s *paymentService) ProcessPayment(ctx context.Context, event model.BookingEvent) error {
	log.Printf("Processing payment for booking %s amount %.2f", event.BookingID, event.TotalAmount)

	// Simulate payment logic (Success mostly)
	status := "SUCCESS"
	transactionID := fmt.Sprintf("txn_%s", event.BookingID) // Dummy txn ID

	// Create payment record
	payment := &model.Payment{
		BookingID:     event.BookingID,
		Amount:        event.TotalAmount,
		Status:        status,
		TransactionID: transactionID,
	}

	if err := s.repo.Create(ctx, payment); err != nil {
		log.Printf("Failed to create payment record: %v", err)
		// If duplicate entry error -> Idempotency handled by DB Unique Constraint?
		// We should publish failed event if it's a real failure
		status = "FAILED"
	}

	// Publish result event
	resultEvent := model.PaymentEvent{
		PaymentID:     payment.ID,
		BookingID:     event.BookingID,
		Amount:        event.TotalAmount,
		Status:        status,
		TransactionID: transactionID,
	}

	topic := "payment.success"
	if status == "FAILED" {
		topic = "payment.failed"
	}

	if err := s.producer.Publish(ctx, topic, event.BookingID, resultEvent); err != nil {
		log.Printf("Failed to publish %s event: %v", topic, err)
		return err
	}

	return nil
}

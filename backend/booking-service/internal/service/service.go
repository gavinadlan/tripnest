package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/gavinadlan/tripnest/backend/booking-service/internal/catalog"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/repository"
)

// Error yang dipetakan handler ke status HTTP. Pakai errors.Is, jangan
// mencocokkan teks error.
var (
	ErrValidation      = errors.New("data booking tidak valid")
	ErrTripNotFound    = errors.New("trip tidak ditemukan")
	ErrTripUnavailable = errors.New("kursi tidak mencukupi")
	ErrForbidden       = errors.New("booking ini bukan milik Anda")
	ErrNotCancellable  = errors.New("booking tidak bisa dibatalkan pada status ini")
)

type BookingService interface {
	CreateBooking(ctx context.Context, userID string, req *model.CreateBookingRequest) (*model.Booking, error)
	GetBooking(ctx context.Context, id string) (*model.Booking, error)
	ListBookings(ctx context.Context, filter repository.ListFilter) ([]model.Booking, error)
	ConfirmBooking(ctx context.Context, bookingID string) error
	CancelBooking(ctx context.Context, bookingID string) error
	CancelOwnBooking(ctx context.Context, userID, bookingID string) error
	ExpirePendingBookings(ctx context.Context) error
}

type bookingService struct {
	repo         repository.BookingRepository
	catalog      *catalog.Client
	expiryWindow time.Duration
}

func NewBookingService(repo repository.BookingRepository, catalogClient *catalog.Client, bookingExpiryMinutes int) BookingService {
	if bookingExpiryMinutes < 10 {
		bookingExpiryMinutes = 10
	}
	return &bookingService{
		repo:         repo,
		catalog:      catalogClient,
		expiryWindow: time.Duration(bookingExpiryMinutes) * time.Minute,
	}
}

// CreateBooking mengambil harga dari katalog, bukan dari request.
//
// userID berasal dari klaim JWT yang sudah divalidasi middleware, bukan
// dari body -- sebelumnya siapa pun bisa membuat booking atas nama orang lain.
func (s *bookingService) CreateBooking(ctx context.Context, userID string, req *model.CreateBookingRequest) (*model.Booking, error) {
	if req.ResourceID == "" {
		return nil, fmt.Errorf("%w: resource_id wajib diisi", ErrValidation)
	}

	quantity := req.Quantity
	if quantity == 0 {
		quantity = 1
	}
	if quantity < 1 || quantity > model.MaxQuantity {
		return nil, fmt.Errorf("%w: quantity harus antara 1 dan %d", ErrValidation, model.MaxQuantity)
	}

	trip, err := s.catalog.GetTrip(ctx, req.ResourceID)
	if err != nil {
		return nil, err
	}
	if trip == nil {
		return nil, ErrTripNotFound
	}
	if trip.Price <= 0 {
		return nil, fmt.Errorf("%w: harga trip tidak valid", ErrValidation)
	}

	// Cek ketersediaan yang cepat dan ramah pengguna. Penentu sebenarnya
	// tetap inventory-service saat mengonsumsi booking.created; cek di sini
	// hanya supaya kasus normal tidak terlanjur masuk ke alur pembayaran.
	if trip.AvailableSlots < quantity {
		return nil, ErrTripUnavailable
	}

	booking := &model.Booking{
		UserID:      userID,
		ResourceID:  trip.ID,
		Quantity:    quantity,
		TotalAmount: roundCurrency(trip.Price * float64(quantity)),
		Status:      model.BookingStatusPendingPayment,
		ExpiresAt:   time.Now().UTC().Add(s.expiryWindow),
	}

	event := &model.BookingCreatedEvent{
		UserID:      booking.UserID,
		ResourceID:  booking.ResourceID,
		Quantity:    booking.Quantity,
		TotalAmount: booking.TotalAmount,
		ExpiresAt:   booking.ExpiresAt.Format(time.RFC3339),
	}

	// Booking dan event outbox-nya masuk dalam satu transaksi. Kalau salah
	// satu gagal, dua-duanya batal -- tidak ada lagi booking tanpa event.
	if err := s.repo.CreateWithEvent(ctx, booking, "booking.created", event); err != nil {
		return nil, err
	}

	return booking, nil
}

func (s *bookingService) GetBooking(ctx context.Context, id string) (*model.Booking, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *bookingService) ListBookings(ctx context.Context, filter repository.ListFilter) ([]model.Booking, error) {
	return s.repo.List(ctx, filter)
}

func (s *bookingService) ConfirmBooking(ctx context.Context, bookingID string) error {
	updated, err := s.repo.UpdateStatusIfCurrent(ctx, bookingID, model.BookingStatusPendingPayment, model.BookingStatusConfirmed)
	if err != nil {
		return err
	}
	if !updated {
		// Sudah dikonfirmasi sebelumnya, atau sudah telanjur kedaluwarsa.
		// Keduanya bukan error bagi consumer; log supaya kasus kedua
		// tetap terlihat saat menelusuri masalah.
		log.Printf("ConfirmBooking tidak mengubah apa pun booking_id=%s", bookingID)
	}
	return nil
}

func (s *bookingService) CancelBooking(ctx context.Context, bookingID string) error {
	updated, err := s.repo.UpdateStatusIfCurrent(ctx, bookingID, model.BookingStatusPendingPayment, model.BookingStatusCancelled)
	if err != nil {
		return err
	}
	if !updated {
		log.Printf("CancelBooking tidak mengubah apa pun booking_id=%s", bookingID)
	}
	return nil
}

// CancelOwnBooking dipakai endpoint HTTP: memastikan booking benar milik
// pemanggil sebelum membatalkan.
func (s *bookingService) CancelOwnBooking(ctx context.Context, userID, bookingID string) error {
	booking, err := s.repo.GetByID(ctx, bookingID)
	if err != nil {
		return err
	}
	if booking == nil {
		return ErrTripNotFound
	}
	if booking.UserID != userID {
		return ErrForbidden
	}
	if booking.Status != model.BookingStatusPendingPayment {
		return ErrNotCancellable
	}

	updated, err := s.repo.UpdateStatusIfCurrent(ctx, bookingID, model.BookingStatusPendingPayment, model.BookingStatusCancelled)
	if err != nil {
		return err
	}
	if !updated {
		return ErrNotCancellable
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

// roundCurrency membulatkan ke 2 desimal supaya nilai yang disimpan cocok
// dengan kolom DECIMAL(14,2) dan tidak menyisakan galat float.
func roundCurrency(v float64) float64 {
	return math.Round(v*100) / 100
}

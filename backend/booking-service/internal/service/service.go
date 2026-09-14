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
	// ConfirmBooking menerima event pembayaran utuh, bukan hanya id,
	// supaya pembayaran yang datang untuk booking yang sudah mati bisa
	// dicatat lengkap dengan nominal dan id transaksinya.
	ConfirmBooking(ctx context.Context, event model.PaymentProcessedEvent) error
	// CancelBooking dipakai consumer (payment.failed, inventory.rejected):
	// kursinya sudah dilepas atau memang tidak pernah tertahan, jadi tidak
	// menerbitkan booking.cancelled.
	CancelBooking(ctx context.Context, bookingID string) error
	// MarkSeatReserved memindahkan booking ke PENDING_PAYMENT setelah
	// kursinya tertahan. Sampai itu terjadi, tidak ada yang boleh
	// menyiapkan pembayaran.
	MarkSeatReserved(ctx context.Context, event model.InventoryReservedEvent) error
	// CancelUnseatedBooking membatalkan booking yang kursinya ditolak.
	CancelUnseatedBooking(ctx context.Context, bookingID string) error
	// ExpireStuckSeatBookings membereskan booking yang tidak pernah
	// mendapat jawaban dari catalog-service.
	ExpireStuckSeatBookings(ctx context.Context) error
	// CancelByUser dan CancelByAdmin adalah pembatalan eksplisit lewat HTTP
	// dan menerbitkan booking.cancelled supaya kursinya kembali beredar.
	CancelByUser(ctx context.Context, userID, bookingID string) error
	CancelByAdmin(ctx context.Context, bookingID string) error
	ExpirePendingBookings(ctx context.Context) error
}

type bookingService struct {
	repo         repository.BookingRepository
	catalog      *catalog.Client
	expiryWindow time.Duration
	seatWindow   time.Duration
}

func NewBookingService(repo repository.BookingRepository, catalogClient *catalog.Client, bookingExpiryMinutes, seatTimeoutSeconds int) BookingService {
	if bookingExpiryMinutes < 10 {
		bookingExpiryMinutes = 10
	}
	// Penahanan kursi normalnya selesai dalam hitungan detik. Batas bawah
	// dipasang supaya sapuan tidak keburu membatalkan booking yang
	// jawabannya sebenarnya sedang di jalan.
	if seatTimeoutSeconds < 30 {
		seatTimeoutSeconds = 30
	}
	return &bookingService{
		repo:         repo,
		catalog:      catalogClient,
		expiryWindow: time.Duration(bookingExpiryMinutes) * time.Minute,
		seatWindow:   time.Duration(seatTimeoutSeconds) * time.Second,
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
		// Kursinya belum tentu ada. Yang menentukan tetap catalog-service;
		// booking menunggu di PENDING_SEAT sampai jawabannya datang.
		Status:    model.BookingStatusPendingSeat,
		ExpiresAt: time.Now().UTC().Add(s.expiryWindow),
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

func (s *bookingService) ConfirmBooking(ctx context.Context, event model.PaymentProcessedEvent) error {
	updated, err := s.repo.UpdateStatusIfCurrent(ctx, event.BookingID, model.BookingStatusPendingPayment, model.BookingStatusConfirmed)
	if err != nil {
		return err
	}
	if updated {
		return nil
	}

	// Transisi gagal berarti booking sudah tidak PENDING_PAYMENT. Perlu
	// dibedakan: event yang terkirim ulang untuk booking yang memang
	// sudah CONFIRMED tidak apa-apa, tapi pembayaran untuk booking yang
	// sudah dibatalkan atau kedaluwarsa berarti ada uang tanpa booking.
	booking, err := s.repo.GetByID(ctx, event.BookingID)
	if err != nil {
		return err
	}
	if booking == nil {
		log.Printf("PERINGATAN: pembayaran sukses untuk booking yang tidak ada booking_id=%s", event.BookingID)
		return nil
	}
	if booking.Status == model.BookingStatusConfirmed {
		return nil
	}

	log.Printf(
		"PERINGATAN: pembayaran sukses untuk booking berstatus %s booking_id=%s payment_id=%s amount=%.2f",
		booking.Status, booking.ID, event.PaymentID, event.Amount,
	)

	orphaned := model.PaymentOrphanedEvent{
		BookingID:     booking.ID,
		PaymentID:     event.PaymentID,
		Amount:        event.Amount,
		BookingStatus: booking.Status,
		TransactionID: event.TransactionID,
	}
	if err := s.repo.AddOutboxEvent(ctx, "payment.orphaned", booking.ID, orphaned); err != nil {
		// Ini justru kasus yang paling tidak boleh hilang diam-diam.
		// Kembalikan error supaya consumer mencoba lagi.
		return fmt.Errorf("gagal mencatat payment.orphaned untuk booking %s: %w", booking.ID, err)
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

// CancelByUser memastikan booking benar milik pemanggil sebelum
// membatalkan.
func (s *bookingService) CancelByUser(ctx context.Context, userID, bookingID string) error {
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
	return s.cancelExplicit(ctx, booking)
}

func (s *bookingService) CancelByAdmin(ctx context.Context, bookingID string) error {
	booking, err := s.repo.GetByID(ctx, bookingID)
	if err != nil {
		return err
	}
	if booking == nil {
		return ErrTripNotFound
	}
	return s.cancelExplicit(ctx, booking)
}

// cancelExplicit membatalkan booking dan mengumumkannya. Booking yang
// sudah CONFIRMED pun boleh dibatalkan: kursinya nyata tertahan, dan
// catalog-service yang akan melepasnya saat mendengar booking.cancelled.
func (s *bookingService) cancelExplicit(ctx context.Context, booking *model.Booking) error {
	// PENDING_SEAT sengaja tidak ikut: kursinya belum tentu tertahan, dan
	// booking.cancelled untuk kursi yang belum ada hanya membingungkan.
	// Booking di keadaan itu selesai sendiri lewat inventory.rejected
	// atau sapuan booking nyangkut, dalam hitungan detik.
	if booking.Status != model.BookingStatusPendingPayment && booking.Status != model.BookingStatusConfirmed {
		return ErrNotCancellable
	}

	updated, err := s.repo.UpdateStatusIfCurrent(ctx, booking.ID, booking.Status, model.BookingStatusCancelled)
	if err != nil {
		return err
	}
	if !updated {
		// Status berubah di antara pembacaan dan penulisan -- misalnya
		// booking keburu kedaluwarsa. Perlakukan sebagai tidak bisa dibatalkan.
		return ErrNotCancellable
	}

	cancelledEvent := model.BookingCancelledEvent{
		BookingID: booking.ID,
		Status:    model.BookingStatusCancelled,
	}
	if err := s.repo.AddOutboxEvent(ctx, "booking.cancelled", booking.ID, cancelledEvent); err != nil {
		// Statusnya sudah berubah; kegagalan di sini hanya menunda
		// kembalinya kursi, jadi jangan gagalkan permintaan user.
		log.Printf("Failed to enqueue booking.cancelled for %s: %v", booking.ID, err)
	}
	return nil
}

// MarkSeatReserved adalah satu-satunya pintu menuju PENDING_PAYMENT.
//
// Kalau transisinya gagal, booking sudah tidak di PENDING_SEAT -- paling
// sering karena keburu dibatalkan sapuan booking nyangkut. Kursinya
// terlanjur tertahan di catalog-service, jadi harus dilepas lagi; tanpa
// ini kursi hilang dari peredaran tanpa ada booking yang memakainya.
func (s *bookingService) MarkSeatReserved(ctx context.Context, event model.InventoryReservedEvent) error {
	updated, err := s.repo.UpdateStatusIfCurrent(ctx, event.BookingID, model.BookingStatusPendingSeat, model.BookingStatusPendingPayment)
	if err != nil {
		return err
	}
	if updated {
		return nil
	}

	booking, err := s.repo.GetByID(ctx, event.BookingID)
	if err != nil {
		return err
	}
	if booking == nil {
		log.Printf("PERINGATAN: kursi tertahan untuk booking yang tidak ada booking_id=%s", event.BookingID)
		return nil
	}
	if booking.Status == model.BookingStatusPendingPayment || booking.Status == model.BookingStatusConfirmed {
		// Event terkirim ulang. Kursinya memang milik booking ini.
		return nil
	}

	log.Printf(
		"Kursi tertahan untuk booking berstatus %s, melepas kembali booking_id=%s",
		booking.Status, booking.ID,
	)
	cancelled := model.BookingCancelledEvent{
		BookingID: booking.ID,
		Status:    model.BookingStatusCancelled,
	}
	if err := s.repo.AddOutboxEvent(ctx, "booking.cancelled", booking.ID, cancelled); err != nil {
		return fmt.Errorf("gagal melepas kursi booking %s: %w", booking.ID, err)
	}
	return nil
}

// CancelUnseatedBooking membatalkan dari PENDING_SEAT, bukan dari
// PENDING_PAYMENT: booking yang kursinya ditolak tidak pernah sampai ke
// keadaan siap bayar.
func (s *bookingService) CancelUnseatedBooking(ctx context.Context, bookingID string) error {
	updated, err := s.repo.UpdateStatusIfCurrent(ctx, bookingID, model.BookingStatusPendingSeat, model.BookingStatusCancelled)
	if err != nil {
		return err
	}
	if !updated {
		log.Printf("CancelUnseatedBooking tidak mengubah apa pun booking_id=%s", bookingID)
	}
	return nil
}

// ExpireStuckSeatBookings membereskan booking yang tidak pernah mendapat
// jawaban dari catalog-service -- consumer-nya mati, event-nya hilang,
// atau Kafka sedang bermasalah.
//
// Tidak ada event pelepasan kursi di sini: menurut catatan kita kursinya
// memang tidak pernah tertahan. Kalau ternyata reservasinya datang
// terlambat, MarkSeatReserved yang melepasnya.
func (s *bookingService) ExpireStuckSeatBookings(ctx context.Context) error {
	cutoff := time.Now().UTC().Add(-s.seatWindow)
	stuckIDs, err := s.repo.CancelStuckSeatBookings(ctx, cutoff)
	if err != nil {
		return err
	}
	for _, bookingID := range stuckIDs {
		log.Printf("PERINGATAN: booking dibatalkan karena kursinya tidak pernah dijawab booking_id=%s", bookingID)
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

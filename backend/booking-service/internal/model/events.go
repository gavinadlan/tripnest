package model

type BookingCreatedEvent struct {
	BookingID   string  `json:"booking_id"`
	UserID      string  `json:"user_id"`
	ResourceID  string  `json:"resource_id"`
	Quantity    int     `json:"quantity"`
	TotalAmount float64 `json:"total_amount"`
	ExpiresAt   string  `json:"expires_at"`
}

// SetBookingID dipanggil repository di dalam transaksi, setelah id booking
// dibangkitkan Postgres tapi sebelum event ditulis ke outbox.
func (e *BookingCreatedEvent) SetBookingID(id string) { e.BookingID = id }

type PaymentProcessedEvent struct {
	PaymentID     string  `json:"payment_id"`
	BookingID     string  `json:"booking_id"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"` // SUCCESS, FAILED
	TransactionID string  `json:"transaction_id"`
}

type BookingExpiredEvent struct {
	BookingID string `json:"booking_id"`
	Status    string `json:"status"`
}

// BookingCancelledEvent hanya diterbitkan untuk pembatalan eksplisit oleh
// user atau admin. catalog-service memakainya untuk melepas kursi, bahkan
// kursi yang reservasinya sudah CONFIRMED.
type BookingCancelledEvent struct {
	BookingID string `json:"booking_id"`
	Status    string `json:"status"`
}

// InventoryRejectedEvent datang dari catalog-service saat kursi ternyata
// tidak cukup. Ini yang menutup loop saga: sebelumnya cabang "kursi habis"
// berakhir di log.Printf dan booking tetap lanjut ke pembayaran.
type InventoryRejectedEvent struct {
	BookingID  string `json:"booking_id"`
	ResourceID string `json:"resource_id"`
	Quantity   int    `json:"quantity"`
	Reason     string `json:"reason"`
}

// InventoryReservedEvent datang dari catalog-service saat kursi benar-benar
// tertahan. Inilah satu-satunya izin untuk mulai menagih: booking-service
// memakainya untuk pindah ke PENDING_PAYMENT, dan payment-service memakainya
// sebagai pemicu pembuatan transaksi Snap.
type InventoryReservedEvent struct {
	BookingID   string  `json:"booking_id"`
	UserID      string  `json:"user_id"`
	ResourceID  string  `json:"resource_id"`
	Quantity    int     `json:"quantity"`
	TotalAmount float64 `json:"total_amount"`
	ExpiresAt   string  `json:"expires_at"`
}

// PaymentOrphanedEvent menandai uang yang masuk untuk booking yang sudah
// tidak bisa dikonfirmasi -- dibatalkan karena kursinya ditolak, atau
// kedaluwarsa. Sebelumnya kasus ini berakhir sebagai no-op diam di
// ConfirmBooking: pembayaran berhasil, booking tetap mati, dan tidak ada
// satu pun jejak yang bisa ditindaklanjuti.
//
// Belum ada consumer-nya: refund otomatis di luar cakupan sprint ini.
// Yang penting sekarang uangnya tercatat dan terlihat, bukan hilang.
type PaymentOrphanedEvent struct {
	BookingID     string  `json:"booking_id"`
	PaymentID     string  `json:"payment_id"`
	Amount        float64 `json:"amount"`
	BookingStatus string  `json:"booking_status"`
	TransactionID string  `json:"transaction_id"`
}

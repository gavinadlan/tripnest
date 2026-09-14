package model

import (
	"errors"
	"strings"
	"time"
)

// DateLayout adalah satu-satunya format tanggal trip yang dipertukarkan
// lewat HTTP maupun event. Read model di search-service menyimpannya
// apa adanya, jadi formatnya tidak boleh berbeda antar jalur.
const DateLayout = "2006-01-02"

// Trip adalah katalog sekaligus kapasitasnya. TotalSlots dan
// AvailableSlots datang dari tabel inventory, yang selalu ditulis dalam
// transaksi yang sama dengan tabel trips.
type Trip struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Destination    string    `json:"destination"`
	Price          float64   `json:"price"`
	Date           string    `json:"date"`
	TotalSlots     int       `json:"total_slots"`
	AvailableSlots int       `json:"available_slots"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TripRequest adalah bentuk payload admin: satu form, satu submit, trip
// dan kapasitasnya sekaligus. Sebelumnya form admin punya field slot tapi
// hanya menulis ke Mongo, sehingga kapasitas tidak pernah benar-benar ada.
type TripRequest struct {
	Title       string  `json:"title"`
	Destination string  `json:"destination"`
	Price       float64 `json:"price"`
	Date        string  `json:"date"`
	TotalSlots  int     `json:"total_slots"`
}

func (r *TripRequest) Normalize() {
	r.Title = strings.TrimSpace(r.Title)
	r.Destination = strings.TrimSpace(r.Destination)
	r.Date = strings.TrimSpace(r.Date)
}

// Validate menolak data yang akan bikin katalog tidak masuk akal.
// Tanggal divalidasi di sini supaya filter pencarian per tanggal di read
// model tidak diam-diam tidak pernah cocok.
func (r *TripRequest) Validate() error {
	if r.Title == "" || r.Destination == "" || r.Date == "" {
		return errors.New("title, destination, dan date wajib diisi")
	}
	if r.Price <= 0 {
		return errors.New("price harus lebih dari 0")
	}
	if r.TotalSlots < 1 {
		return errors.New("total_slots harus minimal 1")
	}
	parsed, err := time.Parse(DateLayout, r.Date)
	if err != nil {
		return errors.New("date harus berformat YYYY-MM-DD")
	}
	if parsed.Before(time.Now().UTC().Truncate(24 * time.Hour)) {
		return errors.New("date tidak boleh di masa lalu")
	}
	return nil
}

// TripUpsertedEvent adalah satu-satunya cara read model tahu isi katalog.
// Payload-nya sengaja lengkap (bukan sekadar id) supaya consumer tidak
// perlu memanggil balik ke sini -- read model tetap bisa dibangun ulang
// walau catalog-service sedang mati.
type TripUpsertedEvent struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Destination    string  `json:"destination"`
	Price          float64 `json:"price"`
	Date           string  `json:"date"`
	TotalSlots     int     `json:"total_slots"`
	AvailableSlots int     `json:"available_slots"`
	UpdatedAt      string  `json:"updated_at"`
}

type TripDeletedEvent struct {
	ID string `json:"id"`
}

// InventoryReservedEvent diterbitkan saat kursi benar-benar tertahan.
//
// Dulu event ini dihapus karena tidak ada yang memakainya. Sekarang ia
// punya tugas yang jelas: menjadi izin untuk mulai menagih. booking-service
// memakainya untuk pindah ke PENDING_PAYMENT dan payment-service untuk
// membuat transaksi Snap. Payloadnya menyalin isi booking.created supaya
// payment-service tidak perlu memanggil balik ke booking-service.
type InventoryReservedEvent struct {
	BookingID   string  `json:"booking_id"`
	UserID      string  `json:"user_id"`
	ResourceID  string  `json:"resource_id"`
	Quantity    int     `json:"quantity"`
	TotalAmount float64 `json:"total_amount"`
	ExpiresAt   string  `json:"expires_at"`
}

// InventoryRejectedEvent menutup loop saga: booking yang tidak kebagian
// kursi harus dibatalkan, bukan diteruskan ke pembayaran. Sebelumnya
// cabang ini berakhir di log.Printf dan booking tetap jalan terus.
type InventoryRejectedEvent struct {
	BookingID  string `json:"booking_id"`
	ResourceID string `json:"resource_id"`
	Quantity   int    `json:"quantity"`
	Reason     string `json:"reason"`
}

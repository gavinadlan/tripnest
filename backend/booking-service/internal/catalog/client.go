// Package catalog adalah klien HTTP ke catalog-service, pemilik trip
// beserta kapasitasnya.
//
// booking-service memakainya untuk mengambil harga resmi dan sisa kursi
// sebuah trip. Harga tidak boleh datang dari request client: sebelumnya
// frontend mengirim total_amount lewat query string, sehingga trip
// berapa pun bisa dibayar seharga apa pun.
package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrUnavailable menandakan katalog tidak bisa dihubungi. Ini berbeda
// dengan "trip tidak ada", dan pemanggil harus membedakannya: yang satu
// masalah kita, yang satu lagi salah input user.
var ErrUnavailable = errors.New("layanan katalog tidak tersedia")

type Trip struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Destination    string  `json:"destination"`
	Price          float64 `json:"price"`
	Date           string  `json:"date"`
	TotalSlots     int     `json:"total_slots"`
	AvailableSlots int     `json:"available_slots"`
}

// tokenMinter menerbitkan token antar-service. Dipenuhi *auth.Verifier;
// dijadikan interface supaya paket ini tidak perlu tahu soal JWT.
type tokenMinter interface {
	ServiceToken(serviceName string) (string, error)
}

type Client struct {
	baseURL string
	http    *http.Client
	minter  tokenMinter
}

func New(baseURL string, minter tokenMinter) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 5 * time.Second},
		minter:  minter,
	}
}

// GetTrip mengembalikan (nil, nil) bila trip tidak ada.
//
// Endpoint yang dipakai adalah /internal/trips/{id}, bukan endpoint
// publik: bentuk responsnya bagian dari kontrak antar-service dan boleh
// berubah tanpa memikirkan browser.
func (c *Client) GetTrip(ctx context.Context, id string) (*Trip, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/internal/trips/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	token, err := c.minter.ServiceToken("booking-service")
	if err != nil {
		return nil, fmt.Errorf("%w: gagal menerbitkan token service: %v", ErrUnavailable, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, nil
	case resp.StatusCode >= http.StatusBadRequest:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("%w: status %d: %s", ErrUnavailable, resp.StatusCode, string(body))
	}

	var trip Trip
	if err := json.NewDecoder(resp.Body).Decode(&trip); err != nil {
		return nil, fmt.Errorf("%w: gagal membaca respons: %v", ErrUnavailable, err)
	}
	if trip.ID == "" {
		return nil, nil
	}
	return &trip, nil
}

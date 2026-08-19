// Package catalog adalah klien HTTP ke service pemilik data trip.
//
// booking-service memakainya untuk mengambil harga resmi sebuah trip.
// Harga tidak boleh datang dari request client: sebelumnya frontend
// mengirim total_amount lewat query string, sehingga trip berapa pun
// bisa dibayar seharga apa pun.
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
	AvailableSlots int     `json:"available_slots"`
}

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// GetTrip mengembalikan (nil, nil) bila trip tidak ada.
func (c *Client) GetTrip(ctx context.Context, id string) (*Trip, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/trips/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

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

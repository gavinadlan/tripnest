package midtrans

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Client struct {
	serverKey  string
	baseURL    string
	httpClient *http.Client
}

type CreateSnapTransactionRequest struct {
	OrderID         string
	GrossAmount     int64
	CustomerDetails CustomerDetails
	// ExpiresAt menyelaraskan masa berlaku Snap dengan masa berlaku booking.
	// Tanpa ini token Snap hidup ~24 jam sementara booking kedaluwarsa
	// dalam 15 menit, sehingga user masih bisa membayar booking yang sudah
	// mati -- uang masuk tapi booking tetap EXPIRED.
	ExpiresAt time.Time
	// ItemName muncul di halaman pembayaran dan struk Midtrans.
	ItemName string
}

type CustomerDetails struct {
	FirstName string
	LastName  string
	Email     string
	Phone     string
}

type CreateSnapTransactionResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

func NewClient(serverKey string, isProduction bool) *Client {
	baseURL := "https://app.sandbox.midtrans.com"
	if isProduction {
		baseURL = "https://app.midtrans.com"
	}

	return &Client{
		serverKey: serverKey,
		baseURL:   baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) CreateSnapTransaction(ctx context.Context, req CreateSnapTransactionRequest) (*CreateSnapTransactionResponse, error) {
	payload := map[string]interface{}{
		"transaction_details": map[string]interface{}{
			"order_id":     req.OrderID,
			"gross_amount": req.GrossAmount,
		},
		"customer_details": map[string]interface{}{
			"first_name": req.CustomerDetails.FirstName,
			"last_name":  req.CustomerDetails.LastName,
			"email":      req.CustomerDetails.Email,
			"phone":      req.CustomerDetails.Phone,
		},
	}

	if itemName := req.ItemName; itemName != "" {
		payload["item_details"] = []map[string]interface{}{{
			"id":       req.OrderID,
			"name":     truncate(itemName, 50), // Midtrans membatasi 50 karakter
			"price":    req.GrossAmount,
			"quantity": 1,
		}}
	}

	if !req.ExpiresAt.IsZero() {
		minutes := int(time.Until(req.ExpiresAt).Minutes())
		if minutes < 1 {
			minutes = 1
		}
		payload["expiry"] = map[string]interface{}{
			"unit":     "minute",
			"duration": minutes,
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal midtrans payload: %w", err)
	}

	url := c.baseURL + "/snap/v1/transactions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build midtrans request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.serverKey+":")))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call midtrans: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading midtrans response: %w", err)
	}
	// Body respons memuat snap token; jangan pernah dicatat utuh di log.
	log.Printf("Midtrans snap response order_id=%s status=%d", req.OrderID, resp.StatusCode)

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("midtrans error status=%d body=%s", resp.StatusCode, string(responseBody))
	}

	var out CreateSnapTransactionResponse
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return nil, fmt.Errorf("failed to decode midtrans response: %w body=%s", err, string(responseBody))
	}

	return &out, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

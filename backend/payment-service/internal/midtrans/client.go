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
	log.Printf("MIDTRANS RAW RESPONSE: %s", string(responseBody))

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("midtrans error status=%d body=%s", resp.StatusCode, string(responseBody))
	}

	var out CreateSnapTransactionResponse
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return nil, fmt.Errorf("failed to decode midtrans response: %w body=%s", err, string(responseBody))
	}

	return &out, nil
}

package charge_api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-tamboon/services/vault_api"
	"net/http"
	"sync"
	"time"
)

type OmiseClient struct {
	SecretKey         string
	BaseURL           string
	vaultClient       vault_api.Client
	chargeRateLimiter *ChargeRateLimiter
}

type ChargeRateLimiter struct {
	tokens     chan struct{}
	interval   time.Duration
	maxTokens  int
	mu         sync.Mutex
	lastRefill time.Time
}

type ChargeRequest struct {
	Name     string
	Amount   int
	Card     string
	CVV      string
	ExpMonth string
	ExpYear  string
}

type ChargeResponse struct {
	Success   bool
	Error     error
	ChargeID  string
	Status    string
	Paid      bool
	Amount    int
	Currency  string
	CreatedAt time.Time
}

type OmiseChargeRequest struct {
	Amount      int    `json:"amount"`
	Currency    string `json:"currency"`
	Card        string `json:"card,omitempty"`
	Customer    string `json:"customer,omitempty"`
	Description string `json:"description,omitempty"`
}

type OmiseChargeResponse struct {
	ID       string `json:"id"`
	Amount   int    `json:"amount"`
	Status   string `json:"status"`
	Paid     bool   `json:"paid"`
	Currency string `json:"currency"`
}

type OmiseError struct {
	Object   string `json:"object"`
	Location string `json:"location"`
	Code     string `json:"code"`
	Message  string `json:"messsage"`
}

type Client interface {
	Charge(req ChargeRequest) ChargeResponse
}

func (e OmiseError) Error() string {
	return fmt.Sprintf("Omise API Error (%s): %s", e.Code, e.Message)
}

func (rl *ChargeRateLimiter) refillTokens() {
	ticker := time.NewTicker(rl.interval / time.Duration(rl.maxTokens))
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		select {
		case rl.tokens <- struct{}{}:
		default:
		}
		rl.mu.Unlock()
	}
}

func (rl *ChargeRateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func NewChargeRateLimiter(maxRequests int, interval time.Duration) *ChargeRateLimiter {
	rl := &ChargeRateLimiter{
		tokens:     make(chan struct{}, maxRequests),
		interval:   interval,
		maxTokens:  maxRequests,
		lastRefill: time.Now(),
	}

	for i := 0; i < maxRequests; i++ {
		rl.tokens <- struct{}{}
	}

	go rl.refillTokens()

	return rl
}

func NewOmiseClient(secretKey, publicKey string, rateLimit int) *OmiseClient {
	vaultClient := vault_api.NewOmiseVaultClient(publicKey, rateLimit)
	baseOmiseURL := "https://api.omise.co/charges"
	return &OmiseClient{
		SecretKey:         secretKey,
		BaseURL:           baseOmiseURL,
		vaultClient:       vaultClient,
		chargeRateLimiter: NewChargeRateLimiter(rateLimit, 5*time.Second), // rateLimit requests per 5 seconds ref with https://docs.omise.co/api-rate-limiting.
	}
}

func (c *OmiseClient) CreateCardToken(cardNumber, holderName, cvv, expMonth, expYear string) (string, error) {
	if c.vaultClient == nil {
		return "", fmt.Errorf("vault client is not initialized - public key required")
	}

	return c.vaultClient.CreateCardToken(cardNumber, holderName, cvv, expMonth, expYear)
}

// Charge creates a charge using the Omise API.
func (c *OmiseClient) Charge(req ChargeRequest) ChargeResponse {
	// Apply rate limiting for charge requests
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.chargeRateLimiter.Wait(ctx); err != nil {
		return ChargeResponse{
			Success: false,
			Error:   fmt.Errorf("rate limit timeout: %w", err),
		}
	}

	// Create token section.
	cardToken, err := c.CreateCardToken(req.Card, req.Name, req.CVV, req.ExpMonth, req.ExpYear)
	if err != nil {
		return ChargeResponse{
			Success: false,
			Error:   fmt.Errorf("failed to create card token: %w", err),
		}
	}

	// Convert to smallest currency unit (bath => satang)
	// Ref with https://docs.omise.co/currency-and-amount.
	convertBathToSatang := req.Amount * 100
	currencyType := "thb"

	// For minimum base on Thailand (minimum: ฿20)
	if convertBathToSatang < 2000 {
		convertBathToSatang = 2000
	}

	// For maximum base on Thailand (maximum: ฿150,000)
	if convertBathToSatang > 15000000 {
		convertBathToSatang = 15000000
	}

	// Prepare Omise request format.
	// Ref with https://docs.omise.co/charges-api.
	omiseReq := OmiseChargeRequest{
		Amount:      convertBathToSatang,
		Currency:    currencyType,
		Card:        cardToken,
		Description: fmt.Sprintf("Donate from  %s", req.Name),
	}

	jsonData, err := json.Marshal(omiseReq)
	if err != nil {
		return ChargeResponse{
			Success: false,
			Error:   fmt.Errorf("failed to marshal request: %w", err),
		}
	}

	// Http for create charges section.
	httpReq, err := http.NewRequest("POST", c.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return ChargeResponse{
			Success: false,
			Error:   fmt.Errorf("failed to create with request: %w", err),
		}
	}

	httpReq.Header.Set("Content-Type", "application/json")

	keyToUse := c.SecretKey
	if keyToUse == "" {
		return ChargeResponse{
			Success: false,
			Error:   fmt.Errorf("private key is required for charge"),
		}
	}

	httpReq.SetBasicAuth(keyToUse, "")

	// Make request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return ChargeResponse{
			Success: false,
			Error:   fmt.Errorf("Request failed: %w", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var omiseErr OmiseError
		if err := json.NewDecoder(resp.Body).Decode(&omiseErr); err != nil {
			return ChargeResponse{
				Success: false,
				Error:   fmt.Errorf("API error with status: %d", resp.StatusCode),
			}
		}

		return ChargeResponse{
			Success: false,
			Error:   omiseErr,
		}
	}

	var omiseRes OmiseChargeResponse
	if err := json.NewDecoder(resp.Body).Decode(&omiseRes); err != nil {
		return ChargeResponse{
			Success: false,
			Error:   fmt.Errorf("failed to decode response from omise: %w", err),
		}
	}

	createdAt, _ := time.Parse(time.RFC3339, "2025-08-10")
	convertSatangToTHB := omiseRes.Amount / 100 // Convert back from smallest unit

	return ChargeResponse{
		Success:   omiseRes.Paid,
		Error:     nil,
		ChargeID:  omiseRes.ID,
		Status:    omiseRes.Status,
		Paid:      omiseRes.Paid,
		Amount:    convertSatangToTHB,
		Currency:  omiseRes.Currency,
		CreatedAt: createdAt,
	}
}

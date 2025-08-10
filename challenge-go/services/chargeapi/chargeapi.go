package chargeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type OmiseClient struct {
	SecretKey   string
	PublicKey   string
	BaseURL     string
	rateLimiter *RateLimiter
}

type RateLimiter struct {
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

type OmiseRequest struct {
	Amount      int                    `json:"amount"`
	Currency    string                 `json:"currency"`
	Card        string                 `json:"card,omitempty"`
	Customer    string                 `json:"customer,omitempty"`
	Description string                 `json:"description,omitempty"`
	Capture     bool                   `json:"capture"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type OmiseResponse struct {
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

type OmiseCardRequest struct {
	Card OmiseCardData `json:"card"`
}

type OmiseCardData struct {
	Name            string `json:"name"`
	Number          string `json:"number"`
	ExpirationMonth string `json:"expiration_month"`
	ExpirationYear  string `json:"expiration_year"`
	SecurityCode    string `json:"security_code"`
}

type OmiseCardResponse struct {
	ID string `json:"id"`
}

type Client interface {
	Charge(req ChargeRequest) ChargeResponse
}

func (e OmiseError) Error() string {
	return fmt.Sprintf("Omise API Error (%s): %s", e.Code, e.Message)
}

func (rl *RateLimiter) refillTokens() {
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

func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func NewRateLimiter(maxRequests int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
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
	baseOmiseURL := "https://api.omise.co"
	return &OmiseClient{
		SecretKey:   secretKey,
		PublicKey:   publicKey,
		BaseURL:     baseOmiseURL,
		rateLimiter: NewRateLimiter(rateLimit, time.Second),
	}
}

func (c *OmiseClient) CreateCardToken(cardNumber, holderName, cvv, expMonth, expYear string) (string, error) {

	// For test
	// plusYear, _ := strconv.Atoi(expYear)

	cardReq := OmiseCardRequest{
		Card: OmiseCardData{
			Name:            holderName,
			Number:          cardNumber,
			ExpirationMonth: expMonth,
			// ExpirationYear:  strconv.Itoa(plusYear + 3),
			ExpirationYear: expYear,
			SecurityCode:   cvv,
		},
	}

	return c.createCardTokenWithData(cardReq)
}

func (c *OmiseClient) createCardTokenWithData(cardReq OmiseCardRequest) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("Rate limit timeout: %w", err)
	}

	formData := url.Values{}
	formData.Set("card[name]", cardReq.Card.Name)
	formData.Set("card[number]", cardReq.Card.Number)
	formData.Set("card[expiration_month]", cardReq.Card.ExpirationMonth)
	formData.Set("card[expiration_year]", cardReq.Card.ExpirationYear)
	formData.Set("card[security_code]", cardReq.Card.SecurityCode)

	// Http for create token from card.
	vaultTokenURL := "https://vault.omise.co/tokens"
	httpReq, err := http.NewRequest("POST", vaultTokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("Failed to create card token request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	publicKey := c.PublicKey
	httpReq.SetBasicAuth(publicKey, "")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("card token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyByte, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("card token API error with status %d : %s", resp.StatusCode, string(bodyByte))
	}

	var cardResp OmiseCardResponse
	if err := json.NewDecoder(resp.Body).Decode(&cardResp); err != nil {
		return "", fmt.Errorf("failed to decode card token response: %w", err)
	}

	return cardResp.ID, nil

}

func (c *OmiseClient) Charge(req ChargeRequest) ChargeResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return ChargeResponse{
			Success: false,
			Error:   fmt.Errorf("rate limit timeout: %w", err),
		}
	}

	cardToken, err := c.CreateCardToken(req.Card, req.Name, req.CVV, req.ExpMonth, req.ExpYear)
	if err != nil {
		return ChargeResponse{
			Success: false,
			Error:   fmt.Errorf("Failed to create card token: %w", err),
		}
	}

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
	omiseReq := OmiseRequest{
		Amount:      convertBathToSatang,
		Currency:    currencyType,
		Card:        cardToken,
		Description: fmt.Sprintf("Donate from  %s", req.Name),
		Capture:     true,
		Metadata: map[string]interface{}{
			"donor_name": req.Name,
			"source":     "tamboon-app",
		},
	}

	// Http for create charges
	jsonData, err := json.Marshal(omiseReq)
	if err != nil {
		return ChargeResponse{
			Success: false,
			Error:   fmt.Errorf("Failed to marshal request: %w", err),
		}
	}

	httpReq, err := http.NewRequest("POST", c.BaseURL+"/charges", bytes.NewBuffer(jsonData))
	if err != nil {
		return ChargeResponse{
			Success: false,
			Error:   fmt.Errorf("Failed to create with request: %w", err),
		}
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.SetBasicAuth(c.SecretKey, "")

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

	var omiseRes OmiseResponse
	if err := json.NewDecoder(resp.Body).Decode(&omiseRes); err != nil {
		return ChargeResponse{
			Success: false,
			Error:   fmt.Errorf("Failed to decode response from omise: %w", err),
		}
	}

	createdAt, _ := time.Parse(time.RFC3339, "2025-08-10")
	convertTHBtoSatang := omiseRes.Amount / 100

	return ChargeResponse{
		Success:   omiseRes.Paid,
		Error:     nil,
		ChargeID:  omiseRes.ID,
		Status:    omiseRes.Status,
		Paid:      omiseRes.Paid,
		Amount:    convertTHBtoSatang,
		Currency:  omiseRes.Currency,
		CreatedAt: createdAt,
	}
}

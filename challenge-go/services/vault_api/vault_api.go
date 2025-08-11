package vault_api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type OmiseVaultClient struct {
	PublicKey        string
	VaultURL         string
	vaultRateLimiter *VaultRateLimiter
}

type VaultRateLimiter struct {
	tokens     chan struct{}
	interval   time.Duration
	maxTokens  int
	mu         sync.Mutex
	lastRefill time.Time
}

type OmiseVaultCardRequest struct {
	Card OmiseVaultCardData `json:"card"`
}

type OmiseVaultCardData struct {
	Name            string `json:"name"`
	Number          string `json:"number"`
	ExpirationMonth string `json:"expiration_month"`
	ExpirationYear  string `json:"expiration_year"`
	SecurityCode    string `json:"security_code"`
}

type OmiseVaultCardResponse struct {
	ID string `json:"id"`
}

type Client interface {
	CreateCardToken(cardNumber, holderName, cvv, expMonth, expYear string) (string, error)
}

func (rl *VaultRateLimiter) refillTokens() {
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

func (rl *VaultRateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func NewOmiseVaultRateLimiter(maxRequests int, interval time.Duration) *VaultRateLimiter {
	rl := &VaultRateLimiter{
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

func NewOmiseVaultClient(publicKey string, rateLimit int) *OmiseVaultClient {
	return &OmiseVaultClient{
		PublicKey:        publicKey,
		VaultURL:         "https://vault.omise.co/tokens",
		vaultRateLimiter: NewOmiseVaultRateLimiter(rateLimit, 5*time.Second), // rateLimit requests per 5 seconds ref with https://docs.omise.co/api-rate-limiting.
	}
}

func (c *OmiseVaultClient) CreateCardToken(cardNumber, holderName, cvv, expMonth, expYear string) (string, error) {
	cardReq := OmiseVaultCardRequest{
		Card: OmiseVaultCardData{
			Name:            holderName,
			Number:          cardNumber,
			ExpirationMonth: expMonth,
			ExpirationYear:  expYear,
			SecurityCode:    cvv,
		},
	}

	return c.createCardTokenWithData(cardReq)
}

// Create and Retrieve information about tokens using the Omise API.
func (c *OmiseVaultClient) createCardTokenWithData(cardReq OmiseVaultCardRequest) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.vaultRateLimiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limit timeout: %w", err)
	}

	// Prepare Omise request format.
	// Ref with https://docs.omise.co/tokens-api.
	jsonData, err := json.Marshal(cardReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal card request: %w", err)
	}

	// Http for create token from card.
	httpReq, err := http.NewRequest("POST", c.VaultURL, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create card token request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	keyToUse := c.PublicKey
	if keyToUse == "" {
		return "", fmt.Errorf("public key is required for card token creation")
	}

	httpReq.SetBasicAuth(keyToUse, "")

	// Make request
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

	var cardResp OmiseVaultCardResponse
	if err := json.NewDecoder(resp.Body).Decode(&cardResp); err != nil {
		return "", fmt.Errorf("failed to decode card token response: %w", err)
	}

	return cardResp.ID, nil

}

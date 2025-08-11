package vault_api

import (
	"context"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	limiter := NewOmiseVaultRateLimiter(5, time.Second)
	if limiter == nil {
		t.Fatal("NewOmiseVaultRateLimiter returned nil")
	}
	if limiter.maxTokens != 5 {
		t.Errorf("Expected maxTokens=5, got %d", limiter.maxTokens)
	}
	if limiter.interval != time.Second {
		t.Errorf("Expected interval=1s, got %v", limiter.interval)
	}
}

func TestRateLimiter_Wait(t *testing.T) {
	limiter := NewOmiseVaultRateLimiter(2, time.Second)
	ctx := context.Background()

	err := limiter.Wait(ctx)
	if err != nil {
		t.Errorf("First request should pass: %v", err)
	}

	err = limiter.Wait(ctx)
	if err != nil {
		t.Errorf("Second request should pass: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err = limiter.Wait(ctx)
	if err == nil {
		t.Error("Third request should timeout")
	}
}

func TestOmiseVaultClient_CreateCardToken_NoPublicKey(t *testing.T) {
	client := &OmiseVaultClient{
		PublicKey:        "",
		VaultURL:         "https://vault.omise.co/tokens",
		vaultRateLimiter: NewOmiseVaultRateLimiter(5, time.Second),
	}

	_, err := client.CreateCardToken("4242424242424242", "Test User", "123", "12", "2025")
	if err == nil {
		t.Error("Expected error when public key is empty")
	}

	if err.Error() != "public key is required for card token creation" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
}

func TestOmiseVaultCardRequest_Structure(t *testing.T) {
	omiseVaultCardData := OmiseVaultCardData{
		Name:            "Test User",
		Number:          "4242424242424242",
		ExpirationMonth: "12",
		ExpirationYear:  "2025",
		SecurityCode:    "123",
	}

	cardReq := OmiseVaultCardRequest{
		Card: omiseVaultCardData,
	}

	if cardReq.Card.Name != "Test User" {
		t.Error("CardRequest structure not working correctly")
	}
}

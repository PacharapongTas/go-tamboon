package charge_api

import (
	"context"
	"strconv"
	"testing"
	"time"
)

func TestNewChargeRateLimiter(t *testing.T) {
	limiter := NewChargeRateLimiter(5, time.Second)
	if limiter == nil {
		t.Fatal("NewChargeRateLimiter returned nil")
	}
	if limiter.maxTokens != 5 {
		t.Errorf("Expected maxTokens=5, got %d", limiter.maxTokens)
	}
	if limiter.interval != time.Second {
		t.Errorf("Expected interval=1s, got %v", limiter.interval)
	}
}

func TestNewChargeRateLimiter_Wait(t *testing.T) {
	limiter := NewChargeRateLimiter(2, time.Second)
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

func TestOmiseClient_Charge_NoPrivateKey(t *testing.T) {
	client := &OmiseClient{
		SecretKey:         "",
		BaseURL:           "https://api.omise.co/charges",
		chargeRateLimiter: NewChargeRateLimiter(5, time.Second),
	}

	result := client.Charge(ChargeRequest{
		Name:     "Test User",
		Amount:   1230,
		CVV:      "4242424242424242",
		ExpMonth: "12",
		ExpYear:  "2025",
	})

	if result.Success {
		t.Fatal("want failure when public key is empty")
	}

	if result.Error == nil || result.Error.Error() !=
		"Failed to create card token: vault client is not initialized - public key required" {
		t.Fatalf("Unexpected error message: %v", result.Error)
	}
}

func TestMockOmiseClient_Charge(t *testing.T) {
	client := NewMockClient()
	success := 0
	fail := 0
	for i := 0; i < 20; i++ {
		res := client.Charge(ChargeRequest{
			Name:     "Test Name" + strconv.Itoa(i),
			Card:     "1234",
			Amount:   100 + i,
			CVV:      "4242424242424242",
			ExpMonth: "12",
			ExpYear:  "2025",
		})
		if res.Success {
			success++
		} else {
			fail++
		}
	}
	if success == 0 || fail == 0 {
		t.Errorf("expected both success and failure, got success=%d fail=%d", success, fail)
	}
}

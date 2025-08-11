package charge_api

import (
	"fmt"
	"math/rand"
	"time"
)

func NewMockClient() Client {
	return &simpleMockClient{}
}

type simpleMockClient struct{}

func (c *simpleMockClient) Charge(req ChargeRequest) ChargeResponse {
	// Test with simple data 80% success rate for main app usage
	if rand.Float32() < 0.8 {
		return ChargeResponse{
			Success:   true,
			ChargeID:  fmt.Sprintf("chrg_mock_%d", time.Now().Unix()),
			Status:    "successful",
			Paid:      true,
			Amount:    req.Amount,
			Currency:  "thb",
			CreatedAt: time.Now(),
		}
	}

	return ChargeResponse{
		Success: false,
		Error:   fmt.Errorf("mock charge failed"),
	}
}

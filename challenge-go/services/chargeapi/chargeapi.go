package chargeapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type OmiseClient struct {
	SecretKey string
	PublicKey string
	BaseURL   string
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

func NewOmiseClient(secretKey, publicKey string) *OmiseClient {
	baseOmiseURL := "https://api.omise.co"
	return &OmiseClient{
		SecretKey: secretKey,
		PublicKey: publicKey,
		BaseURL:   baseOmiseURL,
	}
}

func (c *OmiseClient) CreateCardToken(cardNumber, holderName, cvv, expMonth, expYear string) (string, error) {
	cardReq := OmiseCardRequest{
		Card: OmiseCardData{
			Name:            holderName,
			Number:          cardNumber,
			ExpirationMonth: expMonth,
			ExpirationYear:  expYear,
			SecurityCode:    cvv,
		},
	}

	return c.createCardTokenWithData(cardReq)
}

func (c *OmiseClient) createCardTokenWithData(cardReq OmiseCardRequest) (string, error) {
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

	cardToken, err := c.CreateCardToken(req.Card, req.Name, req.CVV, req.ExpMonth, req.ExpYear)
	if err != nil {
		return ChargeResponse{
			Success: false,
			Error:   fmt.Errorf("Failed to create card token: %w", err),
		}
	}

	convertSatangToBath := req.Amount * 100
	currencyType := "thb"

	// Prepare Omise request format.
	omiseReq := OmiseRequest{
		Amount:      convertSatangToBath,
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

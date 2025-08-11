package main

import (
	"bytes"
	"go-tamboon/cipher"
	"go-tamboon/services/charge_api"
	"go-tamboon/services/csvparser"
	"os"
	"path/filepath"
	"testing"
)

// Mock Data from CSV Column => (Name,AmountSubunits,CCNumber,CVV,ExpMonth,ExpYear)
const mockCSVData = `Luke Skywalker,1000,4242424242424242,123,12,2025
Leia Organa,2000,4000056655665556,456,01,2026
Han Solo,1500,5555555555554444,789,06,2024`

func TestMainWithMockDataCSV(t *testing.T) {
	// Create temporary encrypted file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.csv.rot128")

	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	writer, err := cipher.NewRot128Writer(file)
	if err != nil {
		t.Fatalf("Failed to create Rot128Writer: %v", err)
	}

	_, err = writer.Write([]byte(mockCSVData))
	if err != nil {
		t.Fatalf("Failed to write encrypted data: %v", err)
	}
	file.Close()

	file, err = os.Open(testFile)
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer file.Close()

	rotReader, err := cipher.NewRot128Reader(file)
	if err != nil {
		t.Fatalf("Failed to create Rot128Reader: %v", err)
	}

	var decBuf bytes.Buffer
	_, err = decBuf.ReadFrom(rotReader)
	if err != nil {
		t.Fatalf("Failed to decrypt data: %v", err)
	}

	records, err := csvparser.ParseCSV(bytes.NewReader(decBuf.Bytes()))
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(records) != 3 {
		t.Errorf("Expected 3 records, got %d", len(records))
	}

	// Test with MockClient
	client := charge_api.NewMockClient()

	// Test data first row.
	if len(records) > 0 {
		req := charge_api.ChargeRequest{
			Name:     records[0].Name,
			Card:     records[0].Card,
			Amount:   records[0].Amount,
			CVV:      records[0].CVV,
			ExpMonth: records[0].ExpMonth,
			ExpYear:  records[0].ExpYear,
		}
		result := client.Charge(req)

		if result.Success && result.Error != nil {
			t.Error("Success result should not have error")
		}
		if !result.Success && result.Error == nil {
			t.Error("Failed result should have error")
		}
	}
}

func TestMockClientBehavior(t *testing.T) {
	client := charge_api.NewMockClient()

	// Test multiple charges to see mix of success/failure
	successCount := 0
	failureCount := 0

	for i := 0; i < 50; i++ {
		req := charge_api.ChargeRequest{
			Name:     "Test User",
			Card:     "4242424242424242",
			Amount:   1000,
			CVV:      "123",
			ExpMonth: "12",
			ExpYear:  "2025",
		}
		result := client.Charge(req)

		if result.Success {
			successCount++
			if result.Error != nil {
				t.Error("Success result should not have error")
			}
		} else {
			failureCount++
			if result.Error == nil {
				t.Error("Failed result should have error")
			}
		}
	}

	if successCount == 0 {
		t.Error("Expected some successful charges")
	}
	if failureCount == 0 {
		t.Error("Expected some failed charges")
	}

	t.Logf("Success: %d, Failure: %d", successCount, failureCount)
}

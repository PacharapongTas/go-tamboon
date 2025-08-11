package summary

import (
	"fmt"
	"go-tamboon/services/charge_api"
	"go-tamboon/services/csvparser"
	"testing"
)

func TestCalculateSummary(t *testing.T) {
	records := []csvparser.DataRecord{
		{Name: "A", Card: "1", Amount: 100, CVV: "123", ExpMonth: "05", ExpYear: "2026"},
		{Name: "B", Card: "2", Amount: 200, CVV: "312", ExpMonth: "06", ExpYear: "2021"},
		{Name: "A", Card: "3", Amount: 300, CVV: "111", ExpMonth: "07", ExpYear: "2026"},
	}
	results := []charge_api.ChargeResponse{
		{Success: true},
		{Success: false},
		{Success: true},
	}
	sum := CalculateTotalSummary(records, results)

	if sum.TotalReceived != 600 {
		t.Errorf("expected total 600, got %d", sum.TotalReceived)
	}

	if sum.SuccessDonated != 400 {
		t.Errorf("expected success 400, got %d", sum.SuccessDonated)
	}

	if sum.FailedDonated != 200 {
		t.Errorf("expected faulty 200, got %d", sum.FailedDonated)
	}

	if sum.AveragePerPerson != 200 {
		t.Errorf("expected avg 200, got %f", sum.AveragePerPerson)
	}

	if len(sum.TopDonors) == 0 || sum.TopDonors[0] != "A" {
		t.Errorf("expected top donor A, got %v", sum.TopDonors)
	}
}

func TestAddComma(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"123", "123"},
		{"1234", "1,234"},
		{"12345", "12,345"},
		{"123456", "123,456"},
		{"1234567", "1,234,567"},
		{"12345678", "12,345,678"},
		{"123456789", "123,456,789"},
		{"-1234", "-1,234"},
		{"0", "0"},
		{"", ""},
	}

	for _, item := range tests {
		t.Run(fmt.Sprintf("input_%s", item.input), func(t *testing.T) {
			result := addComma(item.input)
			if result != item.expected {
				t.Errorf("addComma(%s) = %s, expected %s", item.input, result, item.expected)
			}
		})
	}
}

func TestFormatBaht(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{0.0, "0.00"},
		{100.0, "100.00"},
		{1000.0, "1,000.00"},
		{1234.56, "1,234.56"},
		{123456.78, "123,456.78"},
		{1234567.89, "1,234,567.89"},
		{0.50, "0.50"},
		{999.99, "999.99"},
	}

	for _, item := range tests {
		t.Run(fmt.Sprintf("input_%.2f", item.input), func(t *testing.T) {
			result := formatBaht(item.input)
			if result != item.expected {
				t.Errorf("formatBaht(%.2f) = %s, expected %s", item.input, result, item.expected)
			}
		})
	}
}

func TestMaxWidth(t *testing.T) {
	tests := []struct {
		input    []string
		expected int
	}{
		{[]string{"a", "bb", "ccc"}, 3},
		{[]string{"test", "testa"}, 5},
		{[]string{"test"}, 4},
		{[]string{}, 0},
		{[]string{"", "a", ""}, 1},
		{[]string{"test long string", "testshort"}, 16},
	}

	for _, item := range tests {
		t.Run(fmt.Sprintf("input_%v", item.input), func(t *testing.T) {
			result := maxWidth(item.input)
			if result != item.expected {
				t.Errorf("maxWidth(%v) = %d, expected %d", item.input, result, item.expected)
			}
		})
	}
}

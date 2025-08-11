package csvparser

import (
	"strings"
	"testing"
)

func TestParseCSV_Normal(t *testing.T) {
	// Column Format: Name,AmountSubunits,CCNumber,CVV,ExpMonth,ExpYear
	csv := "Luke Skywalker,1000,4242424242424242,123,12,2025\nLeia Organa,2000,4000056655665556,456,01,2026"
	records, err := ParseCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Name != "Luke Skywalker" || records[0].Amount != 1000 || records[0].Card != "4242424242424242" || records[0].CVV != "123" || records[0].ExpMonth != "12" || records[0].ExpYear != "2025" {
		t.Errorf("unexpected record 0 values: %+v", records[0])
	}
	if records[1].Name != "Leia Organa" || records[1].Amount != 2000 || records[1].Card != "4000056655665556" || records[1].CVV != "456" || records[1].ExpMonth != "01" || records[1].ExpYear != "2026" {
		t.Errorf("unexpected record 1 values: %+v", records[1])
	}
}

func TestParseCSV_Empty(t *testing.T) {
	records, err := ParseCSV(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

package csvparser

import (
	"encoding/csv"
	"io"
	"strconv"
)

type DataRecord struct {
	Name     string
	Amount   int
	Card     string
	CVV      string
	ExpMonth string
	ExpYear  string
}

func ParseCSV(r io.Reader) ([]DataRecord, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	wrongFormat := 3
	var records []DataRecord

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			continue
		}

		if len(row) < wrongFormat {
			continue
		}

		amount, err := strconv.Atoi(row[1])
		if err != nil {
			amount = 0
		}

		records = append(records, DataRecord{
			Name:     row[0],
			Amount:   amount,
			Card:     row[2],
			CVV:      row[3],
			ExpMonth: row[4],
			ExpYear:  row[5],
		})
	}

	return records, nil
}

package summary

import (
	"fmt"
	"go-tamboon/services/chargeapi"
	"go-tamboon/services/csvparser"
	"sort"
)

type Summary struct {
	TotalReceived    int
	SuccessDonated   int
	FaiedDonated     int
	AveragePerPerson float64
	TopDonors        []string
}

type Donor struct {
	Name  string
	Total int
}

func CalculateTotalSummary(records []csvparser.DataRecord, results []chargeapi.ChargeResponse) Summary {
	total := 0
	success := 0
	fault := 0
	donorTotals := make(map[string]int)

	for idx, rec := range records {
		total += rec.Amount
		if results[idx].Success {
			success += rec.Amount
			donorTotals[rec.Name] += rec.Amount
		} else {
			fault += rec.Amount
		}
	}

	average := 0.0
	if len(records) > 0 {
		average = float64(total) / float64(len(records))
	}

	var donors []Donor
	for name, amount := range donorTotals {
		donors = append(donors, Donor{
			Name:  name,
			Total: amount,
		})
	}

	sort.Slice(donors, func(i, j int) bool {
		return donors[i].Total > donors[j].Total
	})

	topDonors := []string{}
	topThree := 3
	for i := 0; i < len(donors) && i < topThree; i++ {
		topDonors = append(topDonors, donors[i].Name)
	}

	return Summary{
		TotalReceived:    total,
		SuccessDonated:   success,
		FaiedDonated:     fault,
		AveragePerPerson: average,
		TopDonors:        topDonors,
	}
}

func PrintSummary(s Summary) {
	fmt.Printf("\n total received: THB %9.2f\n", float64(s.TotalReceived))
	fmt.Printf(" successfully donated: %9.2f\n", float64(s.SuccessDonated))
	fmt.Printf(" faulty donation: %9.2f\n", float64(s.FaiedDonated))
	fmt.Printf("\n average per person: %9.2f\n", float64(s.AveragePerPerson))
	fmt.Printf(" top donors: \n")

	for idx, donorName := range s.TopDonors {
		if idx != 0 {
			fmt.Printf("%s\n", donorName)
		}
	}

}

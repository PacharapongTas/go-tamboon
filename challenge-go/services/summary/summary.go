package summary

import (
	"fmt"
	"go-tamboon/services/charge_api"
	"go-tamboon/services/csvparser"
	"sort"
	"strings"
)

type Summary struct {
	TotalReceived    int
	SuccessDonated   int
	FailedDonated    int
	AveragePerPerson float64
	TopDonors        []string
}

type Donor struct {
	Name  string
	Total int
}

func CalculateTotalSummary(records []csvparser.DataRecord, results []charge_api.ChargeResponse) Summary {
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
		FailedDonated:    fault,
		AveragePerPerson: average,
		TopDonors:        topDonors,
	}
}

func addComma(s string) string {
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	n := len(s)
	if n <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	pre := n % 3
	if pre == 0 {
		pre = 3
	}
	b.WriteString(s[:pre])
	for i := pre; i < n; i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	out := b.String()
	if neg {
		return "-" + out
	}
	return out
}

func formatBaht(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	dot := strings.LastIndexByte(s, '.')
	intPart, fracPart := s[:dot], s[dot:]
	return addComma(intPart) + fracPart
}

func maxWidth(vals []string) int {
	max := 0
	for _, s := range vals {
		length := len(s)
		if length > max {
			max = length
		}
	}
	return max
}

func PrintSummary(s Summary) {
	labels := []string{
		" total received:",
		" successfully donated:",
		" faulty donation:",
		" average per person:",
		" top donors:",
	}
	labelWidth := maxWidth(labels)

	summaryValue := []string{
		formatBaht(float64(s.TotalReceived)),
		formatBaht(float64(s.SuccessDonated)),
		formatBaht(float64(s.FailedDonated)),
		formatBaht(float64(s.AveragePerPerson)),
	}

	summaryValueWidth := maxWidth(summaryValue)

	fmt.Printf("%*s THB %*s\n", labelWidth, labels[0], summaryValueWidth, summaryValue[0])
	fmt.Printf("%*s THB %*s\n", labelWidth, labels[1], summaryValueWidth, summaryValue[1])
	fmt.Printf("%*s THB %*s\n", labelWidth, labels[2], summaryValueWidth, summaryValue[2])
	fmt.Println()
	fmt.Printf("%*s THB %*s\n", labelWidth, labels[3], summaryValueWidth, summaryValue[3])

	if len(s.TopDonors) == 0 {
		fmt.Printf("%*s %s \n", labelWidth, labels[4], "-")
		return
	}

	fmt.Printf("%*s %s \n", labelWidth, labels[4], s.TopDonors[0])
	for _, donorName := range s.TopDonors[1:] {
		fmt.Printf("%*s %*s\n", labelWidth, "", summaryValueWidth, donorName)
	}
}

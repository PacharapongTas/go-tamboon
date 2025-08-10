package main

import (
	"bytes"
	"fmt"
	"go-tamboon/cipher"
	"go-tamboon/config"
	"go-tamboon/services/chargeapi"
	"go-tamboon/services/csvparser"
	"go-tamboon/services/summary"
	"io"
	"os"
	"sync"
)

func main() {
	filepath := "./data/fng.1000.csv.rot128"
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Println("Error when opening file: ", err)
		os.Exit(1)
	}
	defer file.Close()

	readerByRot, err := cipher.NewRot128Reader(file)
	if err != nil {
		fmt.Println("Error creating new RotReader: ", err)
		os.Exit(1)
	}
	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, readerByRot)
	if err != nil {
		fmt.Println("Error decrypting file: ", err)
		os.Exit(1)
	}

	// fmt.Println("-- Data After Decode --")
	// fmt.Println(buffer.String())

	records, err := csvparser.ParseCSV(bytes.NewReader(buffer.Bytes()))
	if err != nil {
		fmt.Println("Error Parsing CSV: ", err)
		os.Exit(1)
	}

	// fmt.Println("records", records)

	cfg := config.LoadConfig()
	var client chargeapi.Client
	client = chargeapi.NewOmiseClient(cfg.ChargeSecretKey, cfg.ChargePublicKey)

	results := make([]chargeapi.ChargeResponse, len(records))
	var waitGroup sync.WaitGroup
	workerCount := 8
	jobs := make(chan int, len(records))

	for w := 0; w < workerCount; w++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for idx := range jobs {
				rec := records[idx]
				result := client.Charge(chargeapi.ChargeRequest{
					Name:     rec.Name,
					Amount:   rec.Amount,
					Card:     rec.Card,
					CVV:      rec.CVV,
					ExpMonth: rec.ExpMonth,
					ExpYear:  rec.ExpYear,
				})

				// if !result.Success {
				// 	fmt.Printf("Charge %d failed: %v\n", idx+1, result.Error)
				// } else {
				// 	fmt.Printf("Charge %d Success: %v\n", idx+1, result.Success)
				// }

				results[idx] = result
				records[idx].Card = ""
			}
		}()
	}

	for i := range records {
		jobs <- i
	}

	close(jobs)
	waitGroup.Wait()

	sum := summary.CalculateTotalSummary(records, results)
	summary.PrintSummary(sum)
}

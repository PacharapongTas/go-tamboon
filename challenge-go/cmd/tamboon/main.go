package main

import (
	"bytes"
	"fmt"
	"go-tamboon/cipher"
	"go-tamboon/config"
	"go-tamboon/services/charge_api"
	"go-tamboon/services/csvparser"
	"go-tamboon/services/summary"
	"io"
	"os"
	"sync"
	"time"
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

	records, err := csvparser.ParseCSV(bytes.NewReader(buffer.Bytes()))
	if err != nil {
		fmt.Println("Error Parsing CSV: ", err)
		os.Exit(1)
	}

	cfg := config.LoadConfig()
	var client charge_api.Client
	client = charge_api.NewOmiseClient(cfg.OmiseSecretKey, cfg.OmisePublicKey, cfg.RateLimit)

	results := make([]charge_api.ChargeResponse, len(records))

	// Process in batches (Size and Delay).
	batchSize := cfg.BatchSize
	batchDelay := cfg.BatchDelay

	// Log for tracking process.
	fmt.Printf("Processing %d records in batches of %d with %v delay between batches\n",
		len(records), batchSize, batchDelay)

	for batchStart := 0; batchStart < len(records); batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > len(records) {
			batchEnd = len(records)
		}

		// Log for tracking process.
		fmt.Printf("Processing batch %d-%d...\n", batchStart+1, batchEnd)

		// Process current batch with workers.
		var waitGroup sync.WaitGroup
		workerCount := 2 // Reduce concurrent workers per batch.
		jobs := make(chan int, batchEnd-batchStart)

		for w := 0; w < workerCount; w++ {
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				for idx := range jobs {
					rec := records[idx]
					result := client.Charge(charge_api.ChargeRequest{
						Name:     rec.Name,
						Amount:   rec.Amount,
						Card:     rec.Card,
						CVV:      rec.CVV,
						ExpMonth: rec.ExpMonth,
						ExpYear:  rec.ExpYear,
					})

					// Log for tracking process.
					if !result.Success {
						fmt.Printf("Charge %d failed: %v\n", idx+1, result.Error)
					} else {
						fmt.Printf("Charge %d Success: %v\n", idx+1, result.Success)
					}

					results[idx] = result
					// Wipe card number from memory
					records[idx].Card = ""
				}
			}()
		}

		for i := batchStart; i < batchEnd; i++ {
			jobs <- i
		}

		close(jobs)
		waitGroup.Wait()

		if batchEnd < len(records) {
			// Log for tracking process.
			fmt.Printf("Waiting %v before next batch...\n", batchDelay)
			time.Sleep(batchDelay)
		}
	}

	// Summary Result.
	sum := summary.CalculateTotalSummary(records, results)
	summary.PrintSummary(sum)
}

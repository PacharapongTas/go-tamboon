package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	OmiseSecretKey string
	OmisePublicKey string
	BatchSize      int
	BatchDelay     time.Duration
	RateLimit      int
}

func LoadConfig() *Config {
	LoadEnvFromFile(".env")

	batchSize, _ := strconv.Atoi(getEnv("BATCH_SIZE", "60"))
	if batchSize <= 0 {
		batchSize = 40
	}

	batchDelayed, _ := strconv.Atoi(getEnv("BATCH_DELAY_SECONDS", "10"))
	if batchDelayed <= 0 {
		batchDelayed = 20
	}

	rateLimit, _ := strconv.Atoi(getEnv("RATE_LIMIT", "3"))
	if rateLimit <= 0 {
		rateLimit = 5
	}

	config := &Config{
		OmiseSecretKey: getEnv("OMISE_SECRET_KEY", ""),
		OmisePublicKey: getEnv("OMISE_PUBLIC_KEY", ""),
		BatchSize:      batchSize,
		BatchDelay:     time.Duration(batchDelayed) * time.Second,
		RateLimit:      rateLimit,
	}

	return config
}

func LoadEnvFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return nil
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}

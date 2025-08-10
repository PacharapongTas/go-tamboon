package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	ChargeSecretKey string
	ChargePublicKey string
}

func LoadConfig() *Config {
	LoadEnvFromFile(".env")

	config := &Config{
		ChargeSecretKey: getEnv("CHARGE_SECRET_KEY", ""),
		ChargePublicKey: getEnv("CHARGE_PUBLIC_KEY", ""),
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

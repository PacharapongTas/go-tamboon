package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	os.Setenv("OMISE_SECRET_KEY", "test_secret")
	os.Setenv("OMISE_PUBLIC_KEY", "test_public")
	os.Setenv("BATCH_SIZE", "60")
	os.Setenv("BATCH_DELAY_SECONDS", "10")
	os.Setenv("RATE_LIMIT", "3")

	config := LoadConfig()

	if config.OmiseSecretKey != "test_secret" {
		t.Errorf("Expected OmiseSecretKey to be 'test_secret', got '%s'", config.OmiseSecretKey)
	}

	if config.OmisePublicKey != "test_public" {
		t.Errorf("Expected OmisePublicKey to be 'test_public', got '%s'", config.OmisePublicKey)
	}

	if config.BatchSize <= 0 {
		t.Errorf("Expected BatchSize to be '60', got '%d'", config.BatchSize)
	}

	if config.BatchDelay <= 0 {
		t.Errorf("Expected BatchDelay to be '10', got '%d'", config.BatchDelay)
	}

	if config.RateLimit <= 0 {
		t.Errorf("Expected RateLimit to be '3', got '%d'", config.RateLimit)
	}

	os.Unsetenv("OMISE_SECRET_KEY")
	os.Unsetenv("OMISE_PUBLIC_KEY")
	os.Unsetenv("BATCH_SIZE")
	os.Unsetenv("BATCH_DELAY_SECONDS")
	os.Unsetenv("RATE_LIMIT")
}

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_VARIABLE", "test_set_value")
	value := getEnv("TEST_VARIABLE", "default")
	if value != "test_set_value" {
		t.Errorf("Expected 'test_value', got '%s'", value)
	}

	value = getEnv("NON_EXISTING_VAR", "default")
	if value != "default" {
		t.Errorf("Expected 'default', got '%s'", value)
	}

	os.Unsetenv("TEST_VARIABLE")
}

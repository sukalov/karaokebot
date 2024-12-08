package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func LoadEnv(requiredVars []string) (map[string]string, error) {
	// Load the .env file
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	envVars := make(map[string]string)

	for _, key := range requiredVars {
		value := os.Getenv(key)
		if value == "" {
			return nil, fmt.Errorf("missing required environment variable: %s", key)
		}
		envVars[key] = value
	}

	return envVars, nil
}

func ConvertToMoscowTime(t time.Time) string {
	moscowLocation, _ := time.LoadLocation("Europe/Moscow")
	return t.In(moscowLocation).Format("15:04:05")
}

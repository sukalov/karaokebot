package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func LoadEnv(requiredVars []string) (map[string]string, error) {
	_ = godotenv.Load()

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
	moscowLocation := time.FixedZone("Moscow Time", 3*60*60)
	return t.In(moscowLocation).Format("15:04:05")
}

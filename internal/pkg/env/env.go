package env

import (
	"os"
	"strconv"
)

func GetEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	return defaultValue
}

func GetEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		i, err := strconv.Atoi(val)
		if err != nil {
			return defaultValue
		}
		return i
	}

	return defaultValue
}

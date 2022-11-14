package environ

import (
	"os"
	"strconv"
	"time"
)

func EnvGetString(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func EnvGetBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if v, err := strconv.ParseBool(value); err == nil {
			return v
		}
	}
	return fallback
}

func EnvGetInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}

	return fallback
}

func EnvGetUint64(key string, fallback uint64) uint64 {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.ParseUint(value, 10, 64); err == nil {
			return i
		}
	}

	return fallback
}

func GetDuration(key string, fallback time.Duration) time.Duration {
	if value, ok := os.LookupEnv(key); ok {
		if t, err := time.ParseDuration(value); err == nil {
			return t
		}
	}
	return fallback
}

package config

import "os"

// env reads a raw environment variable. Central choke point so no other file
// touches os.Getenv directly — mirrors Laravel's env() helper.
func env(key string) string {
	return os.Getenv(key)
}

// envDefault returns the variable or a fallback when unset/empty.
func envDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

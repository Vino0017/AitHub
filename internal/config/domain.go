package config

import "os"

// GetDomain returns the configured domain from environment or placeholder
func GetDomain() string {
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "https://your-domain.com"
	}
	return domain
}

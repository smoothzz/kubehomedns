package config

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

var (
	CloudflareAPIKey string
	CloudflareZoneID string
)

func Load(logger *zap.Logger) error {
	CloudflareAPIKey = os.Getenv("CLOUDFLARE_API_KEY")
	if CloudflareAPIKey == "" {
		logger.Error("Environment variable CLOUDFLARE_API_KEY not set")
		return fmt.Errorf("missing CLOUDFLARE_API_KEY environment variable")
	}

	CloudflareZoneID = os.Getenv("CLOUDFLARE_ZONE_ID")
	if CloudflareZoneID == "" {
		logger.Error("Environment variable CLOUDFLARE_ZONE_ID not set")
		return fmt.Errorf("missing CLOUDFLARE_ZONE_ID environment variable")
	}

	return nil
}

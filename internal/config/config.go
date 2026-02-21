package config

import (
	"fmt"
	"time"
)

type Config struct {
	Region           string
	TTL              time.Duration
	InstanceType     string
	TailscaleAuthKey string
	TailscaleAPIKey  string
	TailscaleTailnet string
}

func (c *Config) Validate() error {
	if c.Region == "" {
		return fmt.Errorf("region is required")
	}
	if c.TTL <= 0 {
		return fmt.Errorf("ttl must be positive")
	}
	if c.InstanceType == "" {
		return fmt.Errorf("instance-type is required")
	}
	if c.TailscaleAuthKey == "" {
		return fmt.Errorf("tailscale-auth-key is required (flag or $TAILSCALE_AUTH_KEY)")
	}
	if c.TailscaleAPIKey == "" {
		return fmt.Errorf("tailscale-api-key is required (flag or $TAILSCALE_API_KEY)")
	}
	if c.TailscaleTailnet == "" {
		return fmt.Errorf("tailscale-tailnet is required (flag or $TAILSCALE_TAILNET)")
	}
	return nil
}

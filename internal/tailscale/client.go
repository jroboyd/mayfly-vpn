package tailscale

import (
	"context"
	"fmt"
	"strings"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

// Client wraps the Tailscale API client.
type Client struct {
	inner *tsclient.Client
}

// NewClient creates a Tailscale API client.
func NewClient(apiKey, tailnet string) *Client {
	return &Client{
		inner: &tsclient.Client{
			APIKey:  apiKey,
			Tailnet: tailnet,
		},
	}
}

// FindDevice searches for a device whose hostname starts with the given prefix.
// Returns the device ID if found.
func (c *Client) FindDevice(ctx context.Context, hostnamePrefix string) (string, error) {
	devices, err := c.inner.Devices().List(ctx)
	if err != nil {
		return "", fmt.Errorf("listing devices: %w", err)
	}

	for _, d := range devices {
		if strings.HasPrefix(d.Hostname, hostnamePrefix) {
			return d.ID, nil
		}
	}

	return "", fmt.Errorf("device with hostname prefix %q not found", hostnamePrefix)
}

// ApproveExitNode enables exit node routes (0.0.0.0/0 and ::/0) for a device.
func (c *Client) ApproveExitNode(ctx context.Context, deviceID string) error {
	routes := []string{"0.0.0.0/0", "::/0"}
	if err := c.inner.Devices().SetSubnetRoutes(ctx, deviceID, routes); err != nil {
		return fmt.Errorf("approving exit node routes: %w", err)
	}
	return nil
}

// RemoveDevice deletes a device from the tailnet by ID.
func (c *Client) RemoveDevice(ctx context.Context, deviceID string) error {
	if err := c.inner.Devices().Delete(ctx, deviceID); err != nil {
		return fmt.Errorf("removing device: %w", err)
	}
	return nil
}

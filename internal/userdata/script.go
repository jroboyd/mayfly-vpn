package userdata

import (
	"encoding/base64"
	"fmt"
)

const hostname = "mayfly-exit"

// Hostname returns the hostname used for the Tailscale device.
func Hostname() string {
	return hostname
}

// Generate returns a base64-encoded user-data script that installs Tailscale
// and joins the tailnet as an exit node.
func Generate(authKey string) string {
	script := fmt.Sprintf(`#!/bin/bash
set -euo pipefail

# Enable IP forwarding
cat >> /etc/sysctl.d/99-tailscale.conf <<SYSCTL
net.ipv4.ip_forward = 1
net.ipv6.conf.all.forwarding = 1
SYSCTL
sysctl -p /etc/sysctl.d/99-tailscale.conf

# Install Tailscale
curl -fsSL https://tailscale.com/install.sh | sh

# Start and connect
systemctl enable --now tailscaled
tailscale up --authkey=%s --advertise-exit-node --hostname=%s
`, authKey, hostname)

	return base64.StdEncoding.EncodeToString([]byte(script))
}

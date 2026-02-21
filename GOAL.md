# Mayfly

Ephemeral VPN infrastructure that self-destructs.

## Goal

Mayfly provisions an EC2 instance, installs and configures Tailscale, and joins it to your tailnet -- giving you a ready-to-use VPN exit node. You specify a time-to-live, and when it expires the instance is terminated and all resources are cleaned up, leaving no trace.

## Core Concepts

- **Ephemeral by design** -- every instance has a mandatory TTL
- **Zero residue** -- on expiry the EC2 instance is terminated, the Tailscale device is removed from the tailnet, and all associated AWS resources are cleaned up
- **Minimal configuration** -- provide your AWS and Tailscale credentials, pick a region, set a TTL, and go

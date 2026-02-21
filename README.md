# Mayfly

Ephemeral VPN exit nodes that self-destruct.

Mayfly provisions an EC2 instance with Tailscale configured as an exit node, runs a countdown timer, then tears everything down — leaving zero residue.

## Prerequisites

- Go 1.21+
- AWS credentials configured (via `~/.aws/credentials`, environment variables, or IAM role)
- A default VPC in the target AWS region
- A [Tailscale](https://tailscale.com) account with:
  - An **auth key** (Settings > Keys > Generate auth key)
  - An **API key** (Settings > Keys > Generate API key)

## Build

```sh
go build -o mayfly .
```

Or install to your `$GOPATH/bin`:

```sh
go install .
```

## Usage

```sh
mayfly up --region us-west-2 --ttl 2h
```

All flags have environment variable fallbacks, so you can set them once and just run `mayfly up`:

```sh
export AWS_REGION=us-west-2
export TAILSCALE_AUTH_KEY=tskey-auth-...
export TAILSCALE_API_KEY=tskey-api-...
export TAILSCALE_TAILNET=user@github
```

### Flags

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--region` | `AWS_REGION` | `us-east-1` | AWS region to launch the instance in |
| `--ttl` | `MAYFLY_TTL` | `1h` | Time to live (e.g. `30m`, `2h`, `4h30m`) |
| `--instance-type` | `MAYFLY_INSTANCE_TYPE` | `t3.micro` | EC2 instance type |
| `--tailscale-auth-key` | `TAILSCALE_AUTH_KEY` | — | Tailscale auth key for the node |
| `--tailscale-api-key` | `TAILSCALE_API_KEY` | — | Tailscale API key for device management |
| `--tailscale-tailnet` | `TAILSCALE_TAILNET` | — | Tailscale tailnet name |

## Lifecycle

1. Looks up the latest Amazon Linux 2023 AMI via SSM
2. Creates a security group allowing Tailscale WireGuard traffic (UDP 41641)
3. Launches an EC2 instance with a user-data script that installs Tailscale and joins your tailnet as an exit node
4. Waits for the instance to reach "running" state and displays its public IP
5. Runs a live countdown timer for the TTL duration
6. On TTL expiry **or** Ctrl+C: removes the device from the tailnet, terminates the instance, and deletes the security group

## Crash Recovery

Mayfly writes a state file to `~/.mayfly/state.json` after provisioning. If the process is killed unexpectedly, the next `mayfly up` will detect the orphaned resources and clean them up before proceeding.

The state file only contains AWS resource identifiers (instance ID, security group ID, region) — no secrets.

## Architecture

```
mayfly/
  main.go                          Entry point
  cmd/
    up.go                          CLI command, flags, env var binding
  internal/
    config/config.go               Config struct + validation
    aws/
      ami.go                       SSM parameter lookup for latest AL2023 AMI
      ec2.go                       Provision (SG + instance), Teardown (terminate + delete SG)
    tailscale/client.go            Find and remove devices from the tailnet
    userdata/script.go             Base64-encoded user-data script for Tailscale setup
    runner/runner.go               Orchestrator: provision -> timer -> teardown
    display/status.go              Colored terminal output and countdown timer
    state/state.go                 Crash recovery state file (read/write/clear)
```

### Key design decisions

- **Default VPC only** — keeps provisioning simple; fails clearly if none exists
- **Teardown order** — waits for instance termination before deleting the security group (can't delete an SG while it's in use)
- **Tailscale removal is best-effort** — if the device never joined the tailnet, logs a warning and continues with AWS cleanup
- **Signal handling** — SIGINT/SIGTERM triggers the same graceful teardown as TTL expiry
- **Cleanup uses `context.Background()`** — teardown always runs to completion even if the original context was cancelled

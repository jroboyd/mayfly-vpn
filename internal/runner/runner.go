package runner

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"

	mayaws "github.com/jamesboyd/mayfly/internal/aws"
	"github.com/jamesboyd/mayfly/internal/config"
	"github.com/jamesboyd/mayfly/internal/display"
	"github.com/jamesboyd/mayfly/internal/state"
	"github.com/jamesboyd/mayfly/internal/tailscale"
	"github.com/jamesboyd/mayfly/internal/userdata"
)

func Run(ctx context.Context, cfg *config.Config) error {
	// Check for orphaned resources from a previous crash.
	if err := cleanupOrphans(ctx, cfg); err != nil {
		return err
	}

	// Set up signal handling — Ctrl+C triggers graceful teardown.
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Load AWS config ---
	display.Status("Loading AWS configuration...")
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.Region))
	if err != nil {
		return fmt.Errorf("loading AWS config: %w", err)
	}

	// --- Lookup AMI ---
	display.Status("Looking up latest Amazon Linux 2023 AMI...")
	amiID, err := mayaws.LookupAMI(ctx, awsCfg)
	if err != nil {
		return err
	}
	display.Success(fmt.Sprintf("AMI: %s", amiID))

	// --- Generate user-data ---
	ud := userdata.Generate(cfg.TailscaleAuthKey)

	// --- Provision ---
	display.Status("Provisioning EC2 instance...")
	res, err := mayaws.Provision(ctx, awsCfg, amiID, cfg.InstanceType, ud)

	// Save state immediately so we can recover if we crash after this point.
	saveState(cfg, res)

	if err != nil {
		display.Error(fmt.Sprintf("Provisioning failed: %v", err))
		display.Status("Cleaning up partial resources...")
		teardown(awsCfg, res, cfg)
		return err
	}

	display.Success("Instance running")
	display.Info("Instance ID:", res.InstanceID)
	display.Info("Public IP:", res.PublicIP)
	display.Info("Security Group:", res.SecurityGroupID)
	display.Info("Region:", cfg.Region)
	display.Info("TTL:", cfg.TTL.String())

	// --- Wait for device to join tailnet and approve exit node ---
	display.Status("Waiting for device to join tailnet...")
	tsClient := tailscale.NewClient(cfg.TailscaleAPIKey, cfg.TailscaleTailnet)
	if deviceID, err := waitForDevice(ctx, tsClient, userdata.Hostname()); err != nil {
		display.Warn(fmt.Sprintf("Could not find device in tailnet: %v", err))
	} else {
		display.Success(fmt.Sprintf("Device joined tailnet (ID: %s)", deviceID))
		display.Status("Approving exit node routes...")
		if err := tsClient.ApproveExitNode(ctx, deviceID); err != nil {
			display.Warn(fmt.Sprintf("Could not approve exit node: %v", err))
		} else {
			display.Success("Exit node approved")
		}
	}

	fmt.Println()

	// --- Countdown ---
	deadline := time.Now().Add(cfg.TTL)
	done := make(chan struct{})

	go func() {
		<-ctx.Done()
		close(done)
	}()

	display.Countdown(deadline, done)

	// --- Teardown ---
	if ctx.Err() != nil {
		fmt.Println()
		display.Warn("Interrupted — tearing down...")
	} else {
		display.Status("TTL expired — tearing down...")
	}

	teardown(awsCfg, res, cfg)
	display.Success("All resources cleaned up")
	return nil
}

func cleanupOrphans(ctx context.Context, cfg *config.Config) error {
	prev, err := state.Load()
	if err != nil {
		display.Warn(fmt.Sprintf("Could not read state file: %v", err))
		return nil
	}
	if prev == nil {
		return nil
	}

	display.Warn("Found orphaned resources from a previous run")
	display.Info("Instance ID:", prev.InstanceID)
	display.Info("Security Group:", prev.SecurityGroupID)
	display.Info("Region:", prev.Region)
	display.Status("Cleaning up orphaned resources...")

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(prev.Region))
	if err != nil {
		return fmt.Errorf("loading AWS config for orphan cleanup: %w", err)
	}

	res := &mayaws.Resources{
		InstanceID:      prev.InstanceID,
		SecurityGroupID: prev.SecurityGroupID,
	}

	teardown(awsCfg, res, cfg)
	display.Success("Orphaned resources cleaned up")
	fmt.Println()
	return nil
}

func saveState(cfg *config.Config, res *mayaws.Resources) {
	s := &state.State{
		Region:          cfg.Region,
		InstanceID:      res.InstanceID,
		SecurityGroupID: res.SecurityGroupID,
	}
	if err := state.Save(s); err != nil {
		display.Warn(fmt.Sprintf("Could not save state file: %v", err))
	}
}

// waitForDevice polls the Tailscale API until the device appears or the context is cancelled.
func waitForDevice(ctx context.Context, tsClient *tailscale.Client, hostname string) (string, error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(3 * time.Minute)

	for {
		deviceID, err := tsClient.FindDevice(ctx, hostname)
		if err == nil {
			return deviceID, nil
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "", fmt.Errorf("timed out waiting for device to join tailnet")
		case <-ticker.C:
		}
	}
}

func teardown(awsCfg aws.Config, res *mayaws.Resources, cfg *config.Config) {
	// Remove device from tailnet (best-effort).
	display.Status("Removing device from tailnet...")
	tsClient := tailscale.NewClient(cfg.TailscaleAPIKey, cfg.TailscaleTailnet)

	ctx := context.Background()
	deviceID, err := tsClient.FindDevice(ctx, userdata.Hostname())
	if err != nil {
		display.Warn(fmt.Sprintf("Device not found in tailnet (may not have joined yet): %v", err))
	} else {
		if err := tsClient.RemoveDevice(ctx, deviceID); err != nil {
			display.Warn(fmt.Sprintf("Failed to remove device: %v", err))
		} else {
			display.Success("Device removed from tailnet")
		}
	}

	// Terminate instance + delete security group.
	display.Status("Terminating EC2 instance...")
	if err := mayaws.Teardown(awsCfg, res); err != nil {
		display.Error(fmt.Sprintf("AWS teardown error: %v", err))
	} else {
		display.Success("Instance terminated and security group deleted")
	}

	// Clear state file after successful teardown.
	if err := state.Clear(); err != nil {
		display.Warn(fmt.Sprintf("Could not clear state file: %v", err))
	}
}

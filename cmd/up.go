package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/jamesboyd/mayfly/internal/config"
	"github.com/jamesboyd/mayfly/internal/runner"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mayfly",
	Short: "Ephemeral VPN exit nodes that self-destruct",
	Long:  "Mayfly provisions an EC2 instance with Tailscale configured as an exit node,\nruns a countdown timer, then tears everything down — leaving zero residue.",
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Spin up an ephemeral VPN exit node",
	RunE:  runUp,
}

func init() {
	upCmd.Flags().String("region", envOrDefault("AWS_REGION", "us-east-1"), "AWS region [$AWS_REGION]")
	upCmd.Flags().Duration("ttl", envDurationOrDefault("MAYFLY_TTL", 1*time.Hour), "Time to live [$MAYFLY_TTL]")
	upCmd.Flags().String("instance-type", envOrDefault("MAYFLY_INSTANCE_TYPE", "t3.micro"), "EC2 instance type [$MAYFLY_INSTANCE_TYPE]")
	upCmd.Flags().String("tailscale-auth-key", os.Getenv("TAILSCALE_AUTH_KEY"), "Tailscale auth key [$TAILSCALE_AUTH_KEY]")
	upCmd.Flags().String("tailscale-api-key", os.Getenv("TAILSCALE_API_KEY"), "Tailscale API key [$TAILSCALE_API_KEY]")
	upCmd.Flags().String("tailscale-tailnet", os.Getenv("TAILSCALE_TAILNET"), "Tailscale tailnet name [$TAILSCALE_TAILNET]")

	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	region, _ := cmd.Flags().GetString("region")
	ttl, _ := cmd.Flags().GetDuration("ttl")
	instanceType, _ := cmd.Flags().GetString("instance-type")
	tsAuthKey, _ := cmd.Flags().GetString("tailscale-auth-key")
	tsAPIKey, _ := cmd.Flags().GetString("tailscale-api-key")
	tsTailnet, _ := cmd.Flags().GetString("tailscale-tailnet")

	cfg := &config.Config{
		Region:           region,
		TTL:              ttl,
		InstanceType:     instanceType,
		TailscaleAuthKey: tsAuthKey,
		TailscaleAPIKey:  tsAPIKey,
		TailscaleTailnet: tsTailnet,
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return runner.Run(cmd.Context(), cfg)
}

func Execute() error {
	return rootCmd.Execute()
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

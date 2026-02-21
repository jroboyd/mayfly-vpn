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
	upCmd.Flags().String("region", "", "AWS region [$AWS_REGION] (default \"us-east-1\")")
	upCmd.Flags().Duration("ttl", 0, "Time to live [$MAYFLY_TTL] (default \"1h\")")
	upCmd.Flags().String("instance-type", "", "EC2 instance type [$MAYFLY_INSTANCE_TYPE] (default \"t3.micro\")")
	upCmd.Flags().String("tailscale-auth-key", "", "Tailscale auth key [$TAILSCALE_AUTH_KEY]")
	upCmd.Flags().String("tailscale-api-key", "", "Tailscale API key [$TAILSCALE_API_KEY]")
	upCmd.Flags().String("tailscale-tailnet", "", "Tailscale tailnet name [$TAILSCALE_TAILNET]")

	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	region := flagOrEnv(cmd, "region", "AWS_REGION", "us-east-1")
	ttl := flagDurationOrEnv(cmd, "ttl", "MAYFLY_TTL", 1*time.Hour)
	instanceType := flagOrEnv(cmd, "instance-type", "MAYFLY_INSTANCE_TYPE", "t3.micro")
	tsAuthKey := flagOrEnv(cmd, "tailscale-auth-key", "TAILSCALE_AUTH_KEY", "")
	tsAPIKey := flagOrEnv(cmd, "tailscale-api-key", "TAILSCALE_API_KEY", "")
	tsTailnet := flagOrEnv(cmd, "tailscale-tailnet", "TAILSCALE_TAILNET", "")

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

// flagOrEnv returns the flag value if explicitly set, otherwise the env var, otherwise the fallback.
func flagOrEnv(cmd *cobra.Command, flag, env, fallback string) string {
	if cmd.Flags().Changed(flag) {
		v, _ := cmd.Flags().GetString(flag)
		return v
	}
	if v := os.Getenv(env); v != "" {
		return v
	}
	return fallback
}

func flagDurationOrEnv(cmd *cobra.Command, flag, env string, fallback time.Duration) time.Duration {
	if cmd.Flags().Changed(flag) {
		v, _ := cmd.Flags().GetDuration(flag)
		return v
	}
	if v := os.Getenv(env); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

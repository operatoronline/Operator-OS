// Package svcctl provides CLI commands for managing Operator-OS services.
package svcctl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/operatoronline/Operator-OS/pkg/services"
)

// NewServicesCommand returns the `operator services` command.
func NewServicesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "services",
		Aliases: []string{"svc"},
		Short:   "Manage Operator-OS managed services (browser, sandbox, repo)",
	}

	cmd.AddCommand(
		newStatusCmd(),
		newStartCmd(),
		newStopCmd(),
		newProfileCmd(),
	)

	return cmd
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show status of all managed services",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := services.NewManager(services.DefaultConfig())

			fmt.Printf("Hardware Profile: %s\n", services.ProfileDescription(mgr.Profile()))
			fmt.Printf("Available: %s\n\n", formatServiceList(mgr.AvailableServices()))

			all := mgr.StatusAll()
			fmt.Printf("%-12s %-12s %-14s %s\n", "SERVICE", "STATE", "CONTAINER", "UPTIME")
			fmt.Printf("%-12s %-12s %-14s %s\n", "-------", "-----", "---------", "------")

			for _, stype := range []services.ServiceType{
				services.ServiceBrowser,
				services.ServiceSandbox,
				services.ServiceRepo,
			} {
				info := all[stype]
				uptime := "-"
				if info.StartedAt != nil {
					uptime = time.Since(*info.StartedAt).Round(time.Second).String()
				}
				container := info.ContainerID
				if container == "" {
					container = "-"
				}
				fmt.Printf("%-12s %-12s %-14s %s\n",
					stype, info.State, container, uptime)
			}

			return nil
		},
	}
}

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <service>",
		Short: "Start a managed service (browser, sandbox, repo)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stype := services.ServiceType(args[0])
			mgr := services.NewManager(services.DefaultConfig())

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			fmt.Printf("Starting %s...\n", stype)
			info, err := mgr.EnsureRunning(ctx, stype)
			if err != nil {
				return fmt.Errorf("failed to start %s: %w", stype, err)
			}

			fmt.Printf("✓ %s is running (container: %s)\n", stype, info.ContainerID)
			return nil
		},
	}
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <service>",
		Short: "Stop a managed service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stype := services.ServiceType(args[0])
			mgr := services.NewManager(services.DefaultConfig())

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			fmt.Printf("Stopping %s...\n", stype)
			if err := mgr.Stop(ctx, stype); err != nil {
				return fmt.Errorf("failed to stop %s: %w", stype, err)
			}

			fmt.Printf("✓ %s stopped\n", stype)
			return nil
		},
	}
}

func newProfileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "profile",
		Short: "Show detected hardware profile and available services",
		Run: func(cmd *cobra.Command, args []string) {
			profile := services.DetectHardwareProfile()
			mgr := services.NewManager(services.DefaultConfig())

			fmt.Printf("Profile:    %s\n", profile)
			fmt.Printf("Details:    %s\n", services.ProfileDescription(profile))
			fmt.Printf("Available:  %s\n", formatServiceList(mgr.AvailableServices()))
			fmt.Printf("\nOverride with: OPERATOR_PROFILE=<nano|standard|pro|scale>\n")
		},
	}
}

func formatServiceList(svcs []services.ServiceType) string {
	if len(svcs) == 0 {
		return "(none — core agent only)"
	}
	names := make([]string, len(svcs))
	for i, s := range svcs {
		names[i] = string(s)
	}
	return strings.Join(names, ", ")
}

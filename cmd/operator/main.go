// Operator - Ultra-lightweight personal AI agent
// Operator OS — github.com/operatoronline/Operator-OS
// License: MIT
//
// Copyright (c) 2026 Operator contributors

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/operatoronline/Operator-OS/cmd/operator/internal"
	"github.com/operatoronline/Operator-OS/cmd/operator/internal/agent"
	"github.com/operatoronline/Operator-OS/cmd/operator/internal/auth"
	"github.com/operatoronline/Operator-OS/cmd/operator/internal/cron"
	"github.com/operatoronline/Operator-OS/cmd/operator/internal/gateway"
	"github.com/operatoronline/Operator-OS/cmd/operator/internal/migrate"
	"github.com/operatoronline/Operator-OS/cmd/operator/internal/onboard"
	"github.com/operatoronline/Operator-OS/cmd/operator/internal/skills"
	"github.com/operatoronline/Operator-OS/cmd/operator/internal/status"
	"github.com/operatoronline/Operator-OS/cmd/operator/internal/svcctl"
	"github.com/operatoronline/Operator-OS/cmd/operator/internal/version"
)

func NewOperatorCommand() *cobra.Command {
	short := fmt.Sprintf("%s operator - Personal AI Assistant v%s\n\n", internal.Logo, internal.GetVersion())

	cmd := &cobra.Command{
		Use:     "operator",
		Short:   short,
		Example: "operator list",
	}

	cmd.AddCommand(
		onboard.NewOnboardCommand(),
		agent.NewAgentCommand(),
		auth.NewAuthCommand(),
		gateway.NewGatewayCommand(),
		status.NewStatusCommand(),
		cron.NewCronCommand(),
		migrate.NewMigrateCommand(),
		skills.NewSkillsCommand(),
		svcctl.NewServicesCommand(),
		version.NewVersionCommand(),
	)

	return cmd
}

const (
	colorBlue = "\033[1;38;2;62;93;185m"
	colorRed  = "\033[1;38;2;213;70;70m"
	banner    = "\r\n" +
		colorBlue + "██████╗ ██╗ ██████╗ ██████╗ " + colorRed + " ██████╗██╗      █████╗ ██╗    ██╗\n" +
		colorBlue + "██╔══██╗██║██╔════╝██╔═══██╗" + colorRed + "██╔════╝██║     ██╔══██╗██║    ██║\n" +
		colorBlue + "██████╔╝██║██║     ██║   ██║" + colorRed + "██║     ██║     ███████║██║ █╗ ██║\n" +
		colorBlue + "██╔═══╝ ██║██║     ██║   ██║" + colorRed + "██║     ██║     ██╔══██║██║███╗██║\n" +
		colorBlue + "██║     ██║╚██████╗╚██████╔╝" + colorRed + "╚██████╗███████╗██║  ██║╚███╔███╔╝\n" +
		colorBlue + "╚═╝     ╚═╝ ╚═════╝ ╚═════╝ " + colorRed + " ╚═════╝╚══════╝╚═╝  ╚═╝ ╚══╝╚══╝\n " +
		"\033[0m\r\n"
)

func main() {
	fmt.Printf("%s", banner)
	cmd := NewOperatorCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

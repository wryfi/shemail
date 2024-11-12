package cli

import (
	"github.com/spf13/cobra"
	"github.com/wryfi/shemail/config"
)

func VersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Who am I, Where did I come from?",
		Long:  `Display version/build information for shemail binary.`,
		Run: func(_ *cobra.Command, _ []string) {
			config.PrintShemailVersion()
		},
	}
}

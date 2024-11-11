package cli

import (
	"github.com/spf13/cobra"
	"github.com/wryfi/shemail/config"
	"github.com/wryfi/shemail/logging"
	"os"
)

var log = &logging.Logger

func init() {
	cobra.OnInitialize(config.InitConfig)
}

func SheMailCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "shemail",
		Short: "shemail: sh email client",
		Long:  `shemail is an imap client for the shell, to help you quickly organize your mailboxes`,
	}
	return command
}

func Execute(cmd *cobra.Command) error {
	cmd.SetOut(os.Stdout)
	cmd.AddCommand(ListFolders())
	cmd.AddCommand(SearchFolder())
	cmd.AddCommand(CountMessagesBySender())
	return cmd.Execute()
}

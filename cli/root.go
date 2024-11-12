package cli

import (
	"context"
	"fmt"
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
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			accountRequest, err := cmd.Flags().GetString("account")
			if err != nil {
				return fmt.Errorf("could not get account name: %v", err)
			}
			log.Debug().Msgf("requested account %s", accountRequest)

			account, err := getAccount(accountRequest)
			if err != nil {
				return fmt.Errorf("failed to find requested account; check your configuration")
			}

			// Store the account in the command's context for subcommands to access
			cmd.SetContext(context.WithValue(cmd.Context(), "account", account))
			return nil
		},
	}
	command.PersistentFlags().StringP("account", "A", "default", "account identifier")
	return command
}

func Execute(cmd *cobra.Command) error {
	cmd.SetOut(os.Stdout)
	cmd.AddCommand(ListFolders())
	cmd.AddCommand(SearchFolder())
	cmd.AddCommand(CountMessagesBySender())
	cmd.AddCommand(CreateFolder())
	cmd.AddCommand(VersionCommand())
	cmd.AddCommand(ConfigurationCommand())
	return cmd.Execute()
}

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

// noAuthAnnotation marks commands that do not connect to a mailbox and so
// should skip account/password resolution.
const noAuthAnnotation = "shemail_no_auth"

func init() {
	cobra.OnInitialize(config.InitConfig)
}

// skipAuth reports whether a command should skip account/password resolution.
func skipAuth(cmd *cobra.Command) bool {
	if cmd.Annotations[noAuthAnnotation] == "true" {
		return true
	}
	switch cmd.Name() {
	case "completion", "help":
		return true
	}
	return false
}

func SheMailCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "shemail",
		Short: "shemail: sh email client",
		Long:  `shemail is an imap client for the shell, to help you quickly organize your mailboxes`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Commands that don't talk to a mailbox (version, config, the
			// cobra builtins) need neither an account nor a password.
			if skipAuth(cmd) {
				return nil
			}

			accountRequest, err := cmd.Flags().GetString("account")
			if err != nil {
				return fmt.Errorf("could not get account name: %v", err)
			}
			log.Debug().Msgf("requested account %s", accountRequest)

			account, err := getAccount(accountRequest)
			if err != nil {
				return fmt.Errorf("failed to find requested account; check your configuration")
			}

			password, err := resolvePassword(account)
			if err != nil {
				return err
			}
			account.Password = password

			// Store the account in the command's context for subcommands to access
			cmd.SetContext(context.WithValue(cmd.Context(), "account", account))
			return nil
		},
	}
	command.PersistentFlags().StringP("account", "A", "default", "account identifier")
	command.PersistentFlags().StringVarP(&config.CfgFile, "config", "c", "", "path to config file")
	return command
}

func Execute(cmd *cobra.Command) error {
	cmd.SetOut(os.Stdout)
	cmd.AddCommand(ListFolders())
	cmd.AddCommand(SearchFolder())
	cmd.AddCommand(CountMessagesBySender())
	cmd.AddCommand(CreateFolder())
	cmd.AddCommand(EmptyTrash())
	cmd.AddCommand(VersionCommand())
	cmd.AddCommand(ConfigurationCommand())
	return cmd.Execute()
}

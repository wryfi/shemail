package cli

import (
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/spf13/cobra"
	"github.com/wryfi/shemail/imaputils"
	"github.com/wryfi/shemail/util"
)

// ListFolders generates a command to print a list of imap folders on terminal
func ListFolders() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"folders"},
		Short:   "print a list of folders in the configured mailbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			account := cmd.Context().Value("account").(imaputils.Account)
			folders, err := imaputils.ListFolders(account)
			if err != nil {
				log.Fatal().Msgf("Error listing folders: %v", err)
			}

			for _, folder := range folders {
				fmt.Println(folder)
			}
			return nil
		},
	}
	return cmd
}

// SearchFolder generates a command to search a folder for messages based on various criteria
func SearchFolder() *cobra.Command {
	var (
		endDate   string
		from      string
		or        bool
		startDate string
		subject   string
		to        string
		unread    bool
		read      bool
		moveTo    string
	)
	cmd := &cobra.Command{
		Use:     "find <folder>",
		Short:   "search the specified folder for messages",
		Aliases: []string{"search"},
		Args:    validateFolderArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			account := cmd.Context().Value("account").(imaputils.Account)
			searchOpts := buildSearchOptions(to, from, subject, startDate, endDate, read, unread)

			var criteria *imap.SearchCriteria
			if or {
				criteria = imaputils.BuildORSearchCriteria(searchOpts)
			} else {
				criteria = imaputils.BuildSearchCriteria(searchOpts)
			}

			messages, err := imaputils.SearchMessages(account, args[0], criteria)
			if err != nil {
				log.Fatal().Msgf("Error searching folder %s: %v", args[0], err)
			}

			table := util.TabulateMessages(messages)
			table.Render()

			if moveTo != "" {
				if util.GetConfirmation(fmt.Sprintf("really move %d messages to %s?", len(messages), moveTo)) {
					err := imaputils.MoveMessages(account, messages, args[0], moveTo, 100)
					if err != nil {
						log.Fatal().Msgf("failed to move messages to %s: %v", moveTo, err)
					}
				} else {
					fmt.Println("operation cancelled")
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&to, "to", "t", "", "find messages to this address")
	cmd.Flags().StringVarP(&from, "from", "f", "", "find messages from this address")
	cmd.Flags().StringVarP(&subject, "subject", "s", "", "match subject")
	cmd.Flags().StringVarP(&startDate, "after", "a", "", "find messages received after date (format: `2006-02-01`)")
	cmd.Flags().StringVarP(&endDate, "before", "b", "", "find messages received before date (format: `2006-02-01`)")
	cmd.Flags().BoolVarP(&unread, "unread", "u", false, "find only unread messages")
	cmd.Flags().BoolVarP(&read, "read", "r", false, "find only read messages")
	cmd.Flags().BoolVarP(&or, "or", "o", false, "OR search criteria instead of AND")
	cmd.Flags().StringVarP(&moveTo, "move", "m", "", "move messages to <folder>")
	return cmd
}

// CountMessagesBySender generates a command to list all the senders represented mailbox by how many messages they sent
func CountMessagesBySender() *cobra.Command {
	var threshold int
	cmd := &cobra.Command{
		Use:   "senders <folder>",
		Short: "print a list of senders in the configured mailbox",
		Args:  validateFolderArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			account := cmd.Context().Value("account").(imaputils.Account)
			data, err := imaputils.CountMessagesBySender(account, args[0], threshold)
			if err != nil {
				log.Fatal().Msgf("Error counting messages: %v", err)
			}
			table := util.TabulateSenders(data)
			table.Render()
			return nil
		},
	}
	cmd.Flags().IntVarP(&threshold, "threshold", "t", 1, "only show senders with at least this many messages")
	return cmd
}

// CreateFolder generates a command to recursively create the requested imap folder in account
func CreateFolder() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mkdir <path>",
		Short: "recursively create imap folder",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				log.Fatal().Msg("you must provide the folder path to create as the first positional argument")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			account := cmd.Context().Value("account").(imaputils.Account)
			if err := imaputils.EnsureFolder(account, args[0]); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

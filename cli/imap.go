package cli

import (
	"fmt"
	imap2 "github.com/emersion/go-imap"
	"github.com/spf13/cobra"
	"github.com/wryfi/shemail/imap"
	"github.com/wryfi/shemail/util"
)

// ListFolders prints a list of imap folders on terminal
func ListFolders() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "folders",
		Short: "print a list of folders in the configured mailbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			account := cmd.Context().Value("account").(imap.Account)
			folders, err := imap.ListFolders(account)
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

// SearchFolder searches a folder for messages based on various criteria
func SearchFolder() *cobra.Command {
	var (
		endDate   string
		from      string
		or        bool
		startDate string
		subject   string
		to        string
		unseen    bool
	)
	cmd := &cobra.Command{
		Use:   "search",
		Short: "search the specified folder for messages",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			account := cmd.Context().Value("account").(imap.Account)
			searchOpts := buildSearchOptions(to, from, subject, startDate, endDate, unseen)

			var criteria *imap2.SearchCriteria
			if or {
				criteria = imap.BuildORSearchCriteria(searchOpts)
			} else {
				criteria = imap.BuildSearchCriteria(searchOpts)
			}

			messages, err := imap.SearchMessages(account, args[0], criteria)
			if err != nil {
				log.Fatal().Msgf("Error searching folder %s: %v", args[0], err)
			}

			table := util.TabulateMessages(messages)

			table.Render()
			return nil
		},
	}
	cmd.Flags().BoolVar(&or, "or", false, "`OR` search criteria instead of `AND`")
	cmd.Flags().BoolVar(&unseen, "unseen", false, "find unseen messages")
	cmd.Flags().StringVar(&from, "from", "", "find messages from this address")
	cmd.Flags().StringVar(&to, "to", "", "find messages to this address")
	cmd.Flags().StringVar(&subject, "subject", "", "find messages with this subject")
	cmd.Flags().StringVar(&startDate, "startDate", "", "find messages sent after this date")
	cmd.Flags().StringVar(&endDate, "endDate", "", "find messages sent before this date")
	return cmd
}

// CountMessagesBySender lists all the senders represented mailbox by how many messages they sent
func CountMessagesBySender() *cobra.Command {
	var threshold int
	cmd := &cobra.Command{
		Use:   "senders",
		Short: "print a list of senders in the configured mailbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			account := cmd.Context().Value("account").(imap.Account)
			data, err := imap.CountMessagesBySender(account, args[0], threshold)
			if err != nil {
				log.Fatal().Msgf("Error counting messages: %v", err)
			}
			table := util.TabulateSenders(data)
			table.Render()
			return nil
		},
	}
	cmd.Flags().IntVar(&threshold, "threshold", 1, "only show senders with at least this many messages")
	return cmd
}

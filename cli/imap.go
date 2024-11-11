package cli

import (
	"fmt"
	imap2 "github.com/emersion/go-imap"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wryfi/shemail/imap"
	"github.com/wryfi/shemail/util"
)

func ListFolders() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "folders",
		Short: "print a list of folders in the configured mailbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			server := viper.GetString("imap.server")
			port := viper.GetInt("imap.port")
			user := viper.GetString("imap.user")
			password := viper.GetString("imap.password")
			serverPort := fmt.Sprintf("%s:%d", server, port)

			log.Debug().Msgf("Listing folders on %s for user %s", serverPort, user)
			folders, err := imap.ListFolders(serverPort, user, password)
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
			searchOpts := buildSearchOptions(to, from, subject, startDate, endDate, unseen)

			var criteria *imap2.SearchCriteria
			if or {
				criteria = imap.BuildORSearchCriteria(searchOpts)
			} else {
				criteria = imap.BuildSearchCriteria(searchOpts)
			}

			messages, err := imap.SearchMessages(args[0], criteria)
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

func buildSearchOptions(to, from, subject, startDate, endDate string, unseen bool) imap.SearchOptions {
	searchOpts := imap.SearchOptions{}

	if to != "" {
		searchOpts.To = util.StringPtr(to)
	}
	if from != "" {
		searchOpts.From = util.StringPtr(from)
	}
	if subject != "" {
		searchOpts.Subject = util.StringPtr(subject)
	}
	if startDate != "" {
		log.Debug().Msgf("Parsing start date: %s", startDate)
		timeDate, err := util.DateFromString(startDate)
		if err != nil {
			log.Fatal().Msgf("Error parsing start date %s: %v", startDate, err)
		}
		searchOpts.StartDate = util.TimePtr(timeDate)
	}
	if endDate != "" {
		log.Debug().Msgf("Parsing end date: %s", endDate)
		timeDate, err := util.DateFromString(endDate)
		if err != nil {
			log.Fatal().Msgf("Error parsing end date %s: %v", endDate, err)
		}
		// Add one day to consider the entire end date
		endTime := timeDate.AddDate(0, 0, 1)
		searchOpts.EndDate = util.TimePtr(endTime)
	}
	if unseen {
		searchOpts.Seen = util.BoolPtr(false)
	} else {
		searchOpts.Seen = util.BoolPtr(true)
	}

	log.Debug().Msgf("Search options built: %+v", searchOpts)

	return searchOpts
}

func CountMessagesBySender() *cobra.Command {
	var threshold int
	cmd := &cobra.Command{
		Use:   "senders",
		Short: "print a list of senders in the configured mailbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := imap.CountMessagesBySender(args[0], threshold)
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

package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wryfi/shemail/imap"
	"github.com/wryfi/shemail/util"
)

func getAccount(identifier string) (imap.Account, error) {
	accounts, err := parseAccounts()
	if err != nil {
		log.Fatal().Msgf("failed to parse imap accounts from config file")
	}
	if identifier == "default" {
		log.Debug().Msgf("looking for default account")
		for _, cfgAccount := range accounts {
			if cfgAccount.Default {
				return cfgAccount, nil
			}
		}
	} else {
		log.Debug().Msgf("looking for %s account", identifier)
		for _, cfgAccount := range accounts {
			if cfgAccount.Name == identifier {
				return cfgAccount, nil
			}
		}
	}
	return imap.Account{}, fmt.Errorf("account %q not found", identifier)
}

func parseAccounts() ([]imap.Account, error) {
	var accounts []imap.Account
	if err := viper.UnmarshalKey("accounts", &accounts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal accounts: %w", err)
	}
	return accounts, nil
}

// buildSearchOptions returns a SearchOptions struct from cobra command parameters
func buildSearchOptions(to, from, subject, startDate, endDate string, seen, unseen bool) imap.SearchOptions {
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
	searchOpts.Seen = util.BoolPtr(seen)
	searchOpts.Unseen = util.BoolPtr(unseen)

	log.Debug().Msgf("Search options built: %s", searchOpts.Serialize())

	return searchOpts
}

func validateFolderArg(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("you must specify a folder as the first positional argument")
	}
	return nil
}

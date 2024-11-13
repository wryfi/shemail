package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wryfi/shemail/imaputils"
	"github.com/wryfi/shemail/util"
)

func getAccount(identifier string) (imaputils.Account, error) {
	accounts, err := parseAccounts()
	if err != nil {
		return imaputils.Account{}, fmt.Errorf("failed to parse imap accounts from config file: %w", err)
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
	return imaputils.Account{}, fmt.Errorf("account %q not found", identifier)
}

func parseAccounts() ([]imaputils.Account, error) {
	var accounts []imaputils.Account
	if err := viper.UnmarshalKey("accounts", &accounts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal accounts: %w", err)
	}
	return accounts, nil
}

// buildSearchOptions returns a SearchOptions struct from cobra command parameters
func buildSearchOptions(to, from, subject, startDate, endDate string, seen, unseen bool) (imaputils.SearchOptions, error) {
	searchOpts := imaputils.SearchOptions{}

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
			return imaputils.SearchOptions{}, fmt.Errorf("error parsing start date %s: %w", startDate, err)
		}
		searchOpts.StartDate = util.TimePtr(timeDate)
	}
	if endDate != "" {
		log.Debug().Msgf("Parsing end date: %s", endDate)
		timeDate, err := util.DateFromString(endDate)
		if err != nil {
			return imaputils.SearchOptions{}, fmt.Errorf("error parsing end date %s: %w", endDate, err)
		}
		searchOpts.EndDate = util.TimePtr(timeDate)
	}
	searchOpts.Seen = util.BoolPtr(seen)
	searchOpts.Unseen = util.BoolPtr(unseen)

	log.Debug().Msgf("Search options built: %s", searchOpts.Serialize())

	return searchOpts, nil
}

func validateFolderArg(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("you must specify a folder as the first positional argument")
	}
	return nil
}

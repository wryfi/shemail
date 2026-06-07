package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wryfi/shemail/imaputils"
	"github.com/wryfi/shemail/util"
	"os"
	"os/exec"
	"runtime"
	"strings"
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

// resolvePassword determines an account's password using, in order of
// precedence: the SHEMAIL_<NAME>_PASSWORD environment variable, a literal
// password from configuration, or the first line of output from the account's
// password_command.
func resolvePassword(account imaputils.Account) (string, error) {
	envVar := passwordEnvVar(account.Name)
	if value := os.Getenv(envVar); value != "" {
		log.Debug().Msgf("using password from %s", envVar)
		return value, nil
	}

	if account.Password != "" {
		return account.Password, nil
	}

	if account.PasswordCommand != "" {
		log.Debug().Msgf("resolving password via password_command for account %q", account.Name)
		output, err := shellCommand(account.PasswordCommand).Output()
		if err != nil {
			return "", fmt.Errorf("password_command failed for account %q: %w", account.Name, err)
		}
		password := firstLine(output)
		if password == "" {
			return "", fmt.Errorf("password_command for account %q produced no output", account.Name)
		}
		return password, nil
	}

	return "", fmt.Errorf("no password configured for account %q: set password, password_command, or the %s environment variable", account.Name, envVar)
}

// shellCommand wraps a password_command in the platform's shell so that pipes,
// quoting, and the like work as the user expects: cmd.exe on Windows, sh
// elsewhere.
func shellCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/c", command)
	}
	return exec.Command("sh", "-c", command)
}

// passwordEnvVar returns the per-account password environment variable name,
// e.g. account "work-mail" -> SHEMAIL_WORK_MAIL_PASSWORD. Characters that are
// not alphanumeric are replaced with underscores.
func passwordEnvVar(name string) string {
	upper := strings.Map(func(letter rune) rune {
		switch {
		case letter >= 'a' && letter <= 'z':
			return letter - ('a' - 'A')
		case letter >= 'A' && letter <= 'Z', letter >= '0' && letter <= '9':
			return letter
		default:
			return '_'
		}
	}, name)
	return "SHEMAIL_" + upper + "_PASSWORD"
}

// firstLine returns the first line of output with any trailing carriage return
// removed, so password_command output works whether or not it has a trailing
// newline.
func firstLine(output []byte) string {
	text := string(output)
	if index := strings.IndexByte(text, '\n'); index >= 0 {
		text = text[:index]
	}
	return strings.TrimRight(text, "\r")
}

func parseAccounts() ([]imaputils.Account, error) {
	var accounts []imaputils.Account
	if err := viper.UnmarshalKey("accounts", &accounts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal accounts: %w", err)
	}
	return accounts, nil
}

// buildSearchOptions returns a SearchOptions struct from cobra command parameters
func buildSearchOptions(to, from, subject, notTo, notFrom, notSubject, startDate, endDate, largerThan, smallerThan string, seen, unseen bool) (imaputils.SearchOptions, error) {
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
	if notTo != "" {
		searchOpts.NotTo = util.StringPtr(notTo)
	}
	if notFrom != "" {
		searchOpts.NotFrom = util.StringPtr(notFrom)
	}
	if notSubject != "" {
		searchOpts.NotSubject = util.StringPtr(notSubject)
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
	if largerThan != "" {
		size, err := util.ParseSize(largerThan)
		if err != nil {
			return imaputils.SearchOptions{}, fmt.Errorf("error parsing larger-than size %s: %w", largerThan, err)
		}
		searchOpts.LargerThan = util.Uint32Ptr(size)
	}
	if smallerThan != "" {
		size, err := util.ParseSize(smallerThan)
		if err != nil {
			return imaputils.SearchOptions{}, fmt.Errorf("error parsing smaller-than size %s: %w", smallerThan, err)
		}
		searchOpts.SmallerThan = util.Uint32Ptr(size)
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

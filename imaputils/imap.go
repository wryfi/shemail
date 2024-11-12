package imaputils

import (
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/wryfi/shemail/logging"
	"sort"
	"strings"
)

var log = &logging.Logger

// Account represents the fields that define an IMAP account
type Account struct {
	Name     string // identifier for the account
	User     string
	Password string
	Server   string
	Port     int
	TLS      bool
	Purge    bool
	Default  bool
}

// Create a type that satisfies the io.Writer interface for IMAP debugging
//type debugWriter struct {
//	logger zerolog.Logger
//}
//
//func (w *debugWriter) Write(p []byte) (n int, err error) {
//	w.logger.Debug().Bytes("imap_data", p).Msg("IMAP protocol")
//	return len(p), nil
//}

// getImapClient returns an authenticated imap client for account
func getImapClient(account Account) (*client.Client, error) {
	var connectionError error
	var imapClient *client.Client
	serverPort := fmt.Sprintf("%s:%d", account.Server, account.Port)

	//debug := &debugWriter{
	//	logger: log.With().Str("component", "imap_protocol").Logger(),
	//}

	if account.TLS {
		imapClient, connectionError = client.DialTLS(serverPort, nil)
	} else {
		imapClient, connectionError = client.Dial(serverPort)
	}
	if connectionError != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", connectionError)
	}

	if err := imapClient.Login(account.User, account.Password); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	//imapClient.SetDebug(debug)

	return imapClient, nil
}

// connectToMailbox returns authenticated imap client connected to mailbox
func connectToMailbox(account Account, mailbox string, readonly bool) (*client.Client, error) {
	imapClient, err := getImapClient(account)
	if err != nil {
		return nil, fmt.Errorf("failed to init imap client: %w", err)
	}

	if _, err := imapClient.Select(mailbox, readonly); err != nil {
		imapClient.Logout()
		return nil, fmt.Errorf("failed to select mailbox: %w", err)
	}

	return imapClient, nil
}

// FormatAddress formats an IMAP address into a human-readable string.
func FormatAddress(address *imap.Address) string {
	//var name, mailbox, host string
	var mailbox, host string
	//if address.PersonalName != "" {
	//	name = address.PersonalName
	//}

	if address.MailboxName != "" && address.HostName != "" {
		mailbox = address.MailboxName
		host = address.HostName
	}

	//if name != "" {
	//	return fmt.Sprintf("%s <%s@%s>", name, mailbox, host)
	//}
	return fmt.Sprintf("%s@%s", mailbox, host)
}

// FormatAddresses formats a slice of IMAP addresses into a comma-separated string.
func FormatAddresses(addresses []*imap.Address) []string {
	formatted := []string{}
	for _, addr := range addresses {
		formatted = append(formatted, FormatAddress(addr))
	}
	return formatted
}

// FormatAddressesCSV formats a slice of IMAP addresses into a comma-separated string.
func FormatAddressesCSV(addresses []*imap.Address) string {
	formatted := FormatAddresses(addresses)
	count := len(formatted)
	if count > 1 {
		joined := strings.Join(formatted[0:1], ", ")
		return fmt.Sprintf("%s (+%d)", joined, count-1)
	}
	return strings.Join(formatted, ", ")
}

// sortMessagesByDate sorts messages in reverse chronological order
func sortMessagesByDate(messages []*imap.Message) {
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].InternalDate.After(messages[j].InternalDate)
	})
}

// createSeqSet should work with UIDs, not sequence numbers
func createSeqSet(messages []*imap.Message) *imap.SeqSet {
	seqSet := new(imap.SeqSet)
	for _, msg := range messages {
		seqSet.AddNum(msg.Uid) // Use UID instead of sequence number
	}
	return seqSet
}

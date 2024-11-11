package imap

import (
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/wryfi/shemail/logging"
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
	Default  bool
}

func GetImapClient(account Account) (*client.Client, error) {
	var connectionError error
	var imapClient *client.Client

	serverPort := fmt.Sprintf("%s:%d", account.Server, account.Port)

	if account.TLS {
		imapClient, connectionError = client.DialTLS(serverPort, nil)
	} else {
		imapClient, connectionError = client.Dial(serverPort)
	}
	if connectionError != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", connectionError)
	}

	// Login
	if err := imapClient.Login(account.User, account.Password); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	return imapClient, nil
}

// ListFolders lists all folders in the IMAP account
func ListFolders(account Account) ([]string, error) {
	// Connect to the server
	imapClient, err := GetImapClient(account)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer imapClient.Logout()

	// List mailboxes (folders)
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- imapClient.List("", "*", mailboxes)
	}()

	var folders []string
	for m := range mailboxes {
		folders = append(folders, m.Name)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	return folders, nil
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

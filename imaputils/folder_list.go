package imaputils

import (
	"fmt"
	"github.com/emersion/go-imap"
)

// ListFolders lists all folders in the IMAP account
func ListFolders(dialer IMAPDialer, account Account) ([]string, error) {
	imapClient, err := getImapClient(dialer, account)
	if err != nil {
		return []string{}, fmt.Errorf("failed to initialize imap client: %w", err)
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

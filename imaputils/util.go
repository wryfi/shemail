package imaputils

import (
	"fmt"
	imap2 "github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"sort"
)

// connectAndSelectMailbox establishes IMAP connection and selects the mailbox
func connectAndSelectMailbox(account Account, mailbox string) (*client.Client, error) {
	imapClient, err := GetImapClient(account)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	if _, err := imapClient.Select(mailbox, true); err != nil {
		imapClient.Logout()
		return nil, fmt.Errorf("failed to select mailbox: %w", err)
	}

	return imapClient, nil
}

// sortMessagesByDate sorts messages in reverse chronological order
func sortMessagesByDate(messages []*imap2.Message) {
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].InternalDate.After(messages[j].InternalDate)
	})
}

package imaputils

import (
	"fmt"
	"github.com/emersion/go-imap"
	"golang.org/x/sync/errgroup"
	"strings"
)

// MoveMessages moves a slice of messages to the specified destination folder.
// It uses concurrent operations to optimize performance for large message sets.
func MoveMessages(dialer IMAPDialer, account Account, messages []*imap.Message, sourceFolder, destFolder string, batchSize int) error {
	// Special case for Gmail trash
	if strings.Contains(account.Server, "gmail.com") && destFolder == "[Gmail]/Trash" {
		return moveToGmailTrash(dialer, account, sourceFolder, messages)
	}

	// Just used for initial checks
	imapClient, err := connectToMailbox(dialer, account, sourceFolder, false)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer imapClient.Logout()

	for _, message := range messages {
		log.Debug().Msgf("%d", message.Uid)
	}

	// Ensure destination folder exists
	if err := EnsureFolder(dialer, account, destFolder); err != nil {
		return err
	}

	if batchSize <= 0 {
		batchSize = 100 // Default batch size if not specified
	}

	// Create batches of messages
	var batches [][]*imap.Message
	for i := 0; i < len(messages); i += batchSize {
		end := i + batchSize
		if end > len(messages) {
			end = len(messages)
		}
		batches = append(batches, messages[i:end])
	}

	// Process batches concurrently with separate connections
	g := new(errgroup.Group)
	for _, batch := range batches {
		batch := batch // Create local variable for goroutine
		g.Go(func() error {
			// Create new connection for this batch
			client, err := connectToMailbox(dialer, account, sourceFolder, false)
			if err != nil {
				return fmt.Errorf("failed to connect to server for batch: %w", err)
			}
			defer client.Logout()

			seqSet := new(imap.SeqSet)
			for _, msg := range batch {
				seqSet.AddNum(msg.Uid)
			}

			if err := client.UidMove(seqSet, destFolder); err != nil {
				return fmt.Errorf("failed to move batch: %w", err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("error moving messages: %w", err)
	}

	// Verification could also use a fresh connection
	verifyClient, err := connectToMailbox(dialer, account, sourceFolder, false)
	if err != nil {
		return fmt.Errorf("failed to connect for verification: %w", err)
	}
	defer verifyClient.Logout()

	for _, msg := range messages {
		seqSet := new(imap.SeqSet)
		seqSet.AddNum(msg.Uid)

		fetch := make(chan *imap.Message, 1)
		done := make(chan error, 1)

		go func() {
			done <- verifyClient.UidFetch(seqSet, []imap.FetchItem{imap.FetchUid}, fetch)
		}()

		messageFound := false
		for range fetch {
			messageFound = true
		}

		if err := <-done; err == nil && messageFound {
			return fmt.Errorf("message UID %d still found in source folder after move", msg.Uid)
		}
	}

	return nil
}

// EnsureFolder checks if a folder exists and creates it if it doesn't.
// It handles nested folders by creating parent folders as needed.
func EnsureFolder(dialer IMAPDialer, account Account, folderName string) error {
	imapClient, err := getImapClient(dialer, account)
	if err != nil {
		return fmt.Errorf("failed to init imap client: %w", err)
	}
	defer imapClient.Logout()

	// List existing folders to check if the destination exists
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)

	go func() {
		done <- imapClient.List("", folderName, mailboxes)
	}()

	exists := false
	for mailbox := range mailboxes {
		if mailbox.Name == folderName {
			exists = true
			break
		}
	}

	if err := <-done; err != nil {
		return err
	}

	if exists {
		return nil
	}

	// If folder doesn't exist, create it and any necessary parent folders
	folders := strings.Split(folderName, "/")
	currentPath := ""

	for i, folder := range folders {
		if i > 0 {
			currentPath += "/"
		}
		currentPath += folder

		// Check if this level exists
		mailboxes := make(chan *imap.MailboxInfo, 10)
		done := make(chan error, 1)

		go func() {
			done <- imapClient.List("", currentPath, mailboxes)
		}()

		exists := false
		for mailbox := range mailboxes {
			if mailbox.Name == currentPath {
				exists = true
				break
			}
		}

		if err := <-done; err != nil {
			return err
		}

		if !exists {
			if err := imapClient.Create(currentPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func moveToGmailTrash(dialer IMAPDialer, account Account, folder string, messages []*imap.Message) error {
	imapClient, err := connectToMailbox(dialer, account, folder, false)
	if err != nil {
		return fmt.Errorf("failed to connect to mailbox: %w", err)
	}
	defer imapClient.Logout()

	seqSet := createSeqSet(messages)

	// First copy to Trash using UID
	if err := imapClient.UidCopy(seqSet, "[Gmail]/Trash"); err != nil {
		return fmt.Errorf("failed to copy messages to trash: %w", err)
	}

	// Then use UID STORE to remove the original folder's label
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}
	if err := imapClient.UidStore(seqSet, item, flags, nil); err != nil {
		return fmt.Errorf("failed to flag messages as deleted: %w", err)
	}

	// Use EXPUNGE to remove messages from original folder
	if err := imapClient.Expunge(nil); err != nil {
		return fmt.Errorf("failed to expunge messages: %w", err)
	}

	return nil
}

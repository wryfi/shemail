package imap

import (
	"fmt"
	"github.com/emersion/go-imap"
	"golang.org/x/sync/errgroup"
	"strings"
)

// MoveMessages moves a slice of messages to the specified destination folder.
// It uses concurrent operations to optimize performance for large message sets.
func MoveMessages(account Account, messages []*imap.Message, destFolder string, batchSize int) error {
	imapClient, err := GetImapClient(account)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer imapClient.Logout()

	// Ensure destination folder exists
	if err := EnsureFolder(account, destFolder); err != nil {
		return err
	}

	if batchSize <= 0 {
		batchSize = 100 // Default batch size if not specified
	}

	// Create batches of message UIDs
	var batches [][]*imap.Message
	for i := 0; i < len(messages); i += batchSize {
		end := i + batchSize
		if end > len(messages) {
			end = len(messages)
		}
		batches = append(batches, messages[i:end])
	}

	// Process batches concurrently
	g := new(errgroup.Group)
	for _, batch := range batches {
		batch := batch // Create local variable for goroutine
		g.Go(func() error {
			// Create sequence set for the batch
			seqSet := new(imap.SeqSet)
			for _, msg := range batch {
				seqSet.AddNum(msg.Uid)
			}

			// Move messages in the batch
			return imapClient.UidMove(seqSet, destFolder)
		})
	}

	return g.Wait()
}

// EnsureFolder checks if a folder exists and creates it if it doesn't.
// It handles nested folders by creating parent folders as needed.
func EnsureFolder(account Account, folderName string) error {
	imapClient := MustGetImapClient(account)
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

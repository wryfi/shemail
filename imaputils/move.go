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
//
// Folder paths are specified with "/" as the separator (the shemail
// convention) and translated to the server's actual hierarchy delimiter, which
// is not always "/" (Dovecot, for example, commonly uses ".").
func EnsureFolder(dialer IMAPDialer, account Account, folderName string) error {
	imapClient, err := getImapClient(dialer, account)
	if err != nil {
		return fmt.Errorf("failed to init imap client: %w", err)
	}
	defer imapClient.Logout()

	delimiter, err := getHierarchyDelimiter(imapClient)
	if err != nil {
		return fmt.Errorf("failed to determine hierarchy delimiter: %w", err)
	}
	if delimiter == "" {
		delimiter = "/"
	}

	// Translate the shemail "/"-separated path into the server's delimiter.
	segments := strings.Split(folderName, "/")
	serverPath := strings.Join(segments, delimiter)

	exists, err := mailboxExists(imapClient, serverPath)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// Folder doesn't exist; create it and any necessary parent folders.
	currentPath := ""
	for index, segment := range segments {
		if index > 0 {
			currentPath += delimiter
		}
		currentPath += segment

		exists, err := mailboxExists(imapClient, currentPath)
		if err != nil {
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

// getHierarchyDelimiter returns the server's mailbox hierarchy delimiter,
// discovered via a LIST with empty reference and mailbox name (RFC 3501).
// Returns an empty string if the server reports no delimiter.
func getHierarchyDelimiter(imapClient IMAPClient) (string, error) {
	mailboxes := make(chan *imap.MailboxInfo, 1)
	done := make(chan error, 1)

	go func() {
		done <- imapClient.List("", "", mailboxes)
	}()

	delimiter := ""
	for mailbox := range mailboxes {
		if mailbox.Delimiter != "" {
			delimiter = mailbox.Delimiter
		}
	}

	if err := <-done; err != nil {
		return "", err
	}
	return delimiter, nil
}

// mailboxExists reports whether a mailbox with exactly the given name exists.
func mailboxExists(imapClient IMAPClient, name string) (bool, error) {
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)

	go func() {
		done <- imapClient.List("", name, mailboxes)
	}()

	exists := false
	for mailbox := range mailboxes {
		if mailbox.Name == name {
			exists = true
		}
	}

	if err := <-done; err != nil {
		return false, err
	}
	return exists, nil
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

package imaputils

import (
	"fmt"
	"github.com/emersion/go-imap"
	"strings"
	"time"
)

// FolderStatus describes a mailbox and its message counts.
type FolderStatus struct {
	Name     string
	Messages uint32
	Unseen   uint32
	// Selectable is false for \Noselect container folders (or folders STATUS
	// failed on), for which message counts are not available.
	Selectable bool
	// Oldest and Newest are the earliest and latest message delivery dates in
	// the folder (zero when empty or unavailable).
	Oldest time.Time
	Newest time.Time
}

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

// ListFoldersWithStatus lists all folders along with their message and unseen
// counts, obtained via IMAP STATUS. Non-selectable container folders
// (\Noselect) are included with Selectable=false and no counts. A STATUS that
// fails for an individual folder is reported as non-selectable rather than
// failing the whole listing.
// When withDates is true, each folder's message date range (Oldest/Newest) is
// computed by scanning every message's internal date — accurate but slower, so
// it is opt-in. When false, only the message and unread counts are populated.
func ListFoldersWithStatus(dialer IMAPDialer, account Account, withDates bool) ([]FolderStatus, error) {
	imapClient, err := getImapClient(dialer, account)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize imap client: %w", err)
	}
	defer imapClient.Logout()

	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- imapClient.List("", "*", mailboxes)
	}()

	var infos []*imap.MailboxInfo
	for mailbox := range mailboxes {
		infos = append(infos, mailbox)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	statusItems := []imap.StatusItem{imap.StatusMessages, imap.StatusUnseen}
	folders := make([]FolderStatus, 0, len(infos))
	for _, info := range infos {
		folder := FolderStatus{Name: info.Name}

		if hasAttribute(info.Attributes, imap.NoSelectAttr) {
			folders = append(folders, folder)
			continue
		}

		status, err := imapClient.Status(info.Name, statusItems)
		if err != nil {
			log.Debug().Msgf("failed to get status for folder %q: %v", info.Name, err)
			folders = append(folders, folder)
			continue
		}

		folder.Selectable = true
		folder.Messages = status.Messages
		folder.Unseen = status.Unseen

		if withDates && status.Messages > 0 {
			oldest, newest, err := folderDateRange(imapClient, info.Name)
			if err != nil {
				log.Debug().Msgf("failed to get date range for folder %q: %v", info.Name, err)
			} else {
				folder.Oldest = oldest
				folder.Newest = newest
			}
		}

		folders = append(folders, folder)
	}

	return folders, nil
}

// folderDateRange selects the given folder read-only and returns the earliest
// and latest message delivery (INTERNALDATE) dates. It returns zero times for
// an empty mailbox. Note this fetches the internal date of every message in the
// folder, so it is the expensive part of a status listing.
func folderDateRange(imapClient IMAPClient, folder string) (time.Time, time.Time, error) {
	mbox, err := imapClient.Select(folder, true)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if mbox.Messages == 0 {
		return time.Time{}, time.Time{}, nil
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddRange(1, mbox.Messages)

	messages := make(chan *imap.Message, 100)
	done := make(chan error, 1)
	go func() {
		done <- imapClient.Fetch(seqSet, []imap.FetchItem{imap.FetchInternalDate}, messages)
	}()

	var oldest, newest time.Time
	for message := range messages {
		date := message.InternalDate
		if date.IsZero() {
			continue
		}
		if oldest.IsZero() || date.Before(oldest) {
			oldest = date
		}
		if newest.IsZero() || date.After(newest) {
			newest = date
		}
	}

	if err := <-done; err != nil {
		return time.Time{}, time.Time{}, err
	}

	return oldest, newest, nil
}

// hasAttribute reports whether the mailbox attribute list contains target
// (case-insensitively).
func hasAttribute(attributes []string, target string) bool {
	for _, attribute := range attributes {
		if strings.EqualFold(attribute, target) {
			return true
		}
	}
	return false
}

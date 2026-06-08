package imaputils

import (
	"fmt"
	"github.com/emersion/go-imap"
	"strings"
)

// FolderStatus describes a mailbox and its message counts.
type FolderStatus struct {
	Name     string
	Messages uint32
	Unseen   uint32
	// Selectable is false for \Noselect container folders (or folders STATUS
	// failed on), for which message counts are not available.
	Selectable bool
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
func ListFoldersWithStatus(dialer IMAPDialer, account Account) ([]FolderStatus, error) {
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
		folders = append(folders, folder)
	}

	return folders, nil
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

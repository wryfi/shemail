package imaputils

import (
	"fmt"
	"github.com/emersion/go-imap"
)

type DeletionStrategy int

var DeletedFolderNames = []string{
	"Deleted Items",
	"Trash",
	"Deleted Messages",
	"[Gmail]/Trash",
}

// DeleteMessages deletes the list of messages based on the account's deletion strategy
func DeleteMessages(account Account, messages []*imap.Message, folder string) error {
	var err error
	if len(messages) == 0 {
		return nil
	}
	if account.Purge {
		log.Debug().Msgf("will purge messages from this folder")
		err = purgeMessages(account, folder, messages)
	} else {
		err = moveToTrash(account, folder, messages)
	}
	if err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}
	return nil
}

// moveToTrash moves a list of messages to a trash/deleted folder
func moveToTrash(account Account, folder string, messages []*imap.Message) error {
	trashFolder, err := findTrashFolder(account)
	if err != nil {
		return fmt.Errorf("failed to find trash folder: %w", err)
	}
	if err := MoveMessages(account, messages, folder, trashFolder, 100); err != nil {
		return fmt.Errorf("failed to move messages from %s to %s: %w", folder, trashFolder, err)
	}
	return nil
}

// findTrashFolder searches account folders for common trash folder names
func findTrashFolder(account Account) (string, error) {
	mailboxes, err := ListFolders(account)
	if err != nil {
		return "", fmt.Errorf("failed to list folders: %w", err)
	}
	for _, folder := range mailboxes {
		for _, trashName := range DeletedFolderNames {
			if folder == trashName {
				return trashName, nil
			}
		}
	}
	return "Deleted Items", nil
}

// purgeMessages permanently deletes a list of messages from a folder
func purgeMessages(account Account, folder string, messages []*imap.Message) error {
	imapClient, err := connectToMailbox(account, folder, false)
	if err != nil {
		return fmt.Errorf("failed to connect to mailbox: %w", err)
	}
	defer imapClient.Logout()

	seqSet := createSeqSet(messages)
	action := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}
	if err := imapClient.UidStore(seqSet, action, flags, nil); err != nil {
		return fmt.Errorf("failed to mark messages as deleted: %w", err)
	}
	if err := imapClient.Expunge(nil); err != nil {
		return fmt.Errorf("failed to expunge messages: %w", err)
	}
	return nil
}

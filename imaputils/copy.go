package imaputils

import (
	"fmt"
	"github.com/emersion/go-imap"
)

// CopyMessages copies the given messages to destFolder (creating it if needed),
// leaving the originals in sourceFolder. The source is opened read-only since
// COPY does not modify it.
func CopyMessages(dialer IMAPDialer, account Account, messages []*imap.Message, sourceFolder, destFolder string) error {
	if len(messages) == 0 {
		return nil
	}

	if err := EnsureFolder(dialer, account, destFolder); err != nil {
		return err
	}

	imapClient, err := connectToMailbox(dialer, account, sourceFolder, true)
	if err != nil {
		return fmt.Errorf("failed to connect to mailbox: %w", err)
	}
	defer imapClient.Logout()

	seqSet := createSeqSet(messages)
	if err := imapClient.UidCopy(seqSet, destFolder); err != nil {
		return fmt.Errorf("failed to copy messages to %s: %w", destFolder, err)
	}

	return nil
}

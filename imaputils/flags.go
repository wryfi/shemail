package imaputils

import (
	"fmt"
	"github.com/emersion/go-imap"
)

// MarkMessages adds or removes the \Seen flag on the given messages in the
// specified folder: seen=true marks them read, seen=false marks them unread.
func MarkMessages(dialer IMAPDialer, account Account, messages []*imap.Message, folder string, seen bool) error {
	if len(messages) == 0 {
		return nil
	}

	imapClient, err := connectToMailbox(dialer, account, folder, false)
	if err != nil {
		return fmt.Errorf("failed to connect to mailbox: %w", err)
	}
	defer imapClient.Logout()

	var operation imap.FlagsOp = imap.RemoveFlags
	if seen {
		operation = imap.AddFlags
	}
	item := imap.FormatFlagsOp(operation, true)
	flags := []interface{}{imap.SeenFlag}

	seqSet := createSeqSet(messages)
	if err := imapClient.UidStore(seqSet, item, flags, nil); err != nil {
		return fmt.Errorf("failed to update \\Seen flag: %w", err)
	}

	return nil
}

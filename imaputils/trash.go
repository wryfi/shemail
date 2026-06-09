package imaputils

import (
	"fmt"
	"github.com/emersion/go-imap"
)

// EmptyFolder permanently deletes every message in the given folder, returning
// the number of messages removed. The caller is responsible for resolving the
// folder name (e.g. via FindTrashFolder) so it can be confirmed before the
// irreversible expunge.
func EmptyFolder(dialer IMAPDialer, account Account, folder string) (int, error) {
	imapClient, err := connectToMailbox(dialer, account, folder, false)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to mailbox: %w", err)
	}
	defer imapClient.Logout()

	// Find every message by UID rather than fetching envelopes we don't need.
	uids, err := imapClient.UidSearch(BuildSearchCriteria(SearchOptions{}))
	if err != nil {
		return 0, fmt.Errorf("failed to search folder %s: %w", folder, err)
	}
	if len(uids) == 0 {
		return 0, nil
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)

	action := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}
	if err := imapClient.UidStore(seqSet, action, flags, nil); err != nil {
		return 0, fmt.Errorf("failed to mark messages as deleted: %w", err)
	}
	if err := imapClient.Expunge(nil); err != nil {
		return 0, fmt.Errorf("failed to expunge folder %s: %w", folder, err)
	}

	return len(uids), nil
}

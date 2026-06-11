package imaputils

import (
	"encoding/json"
	"fmt"
	"github.com/emersion/go-imap"
	"time"
)

// SearchOptions represents the optional search parameters
type SearchOptions struct {
	To      *string // Optional To address
	From    *string // Optional From address
	NotTo   *string // Optional negated To address (must NOT match)
	NotFrom *string // Optional negated From address (must NOT match)
	// Subject/NotSubject are matched client-side (see FilterBySubject), not via
	// server-side SEARCH. A message is kept if its subject matches ANY Subject
	// pattern and NONE of the NotSubject patterns.
	Subject     []string
	NotSubject  []string
	StartDate   *time.Time // Optional start date
	EndDate     *time.Time // Optional end date
	Seen        *bool      // Optional seen flag
	Unseen      *bool      // Optional unseen flag
	LargerThan  *uint32    // Optional minimum size in bytes (exclusive)
	SmallerThan *uint32    // Optional maximum size in bytes (exclusive)
	// SubjectRegex treats Subject/NotSubject as regular expressions rather than
	// case-insensitive substrings.
	SubjectRegex bool
}

// Serialize serializes SearchOptions to json
func (opts SearchOptions) Serialize() string {
	jsonBytes, _ := json.MarshalIndent(opts, "", "  ")
	return string(jsonBytes)
}

// SearchMessages performs a search for messages in the specified mailbox using given criteria
func SearchMessages(dialer IMAPDialer, account Account, mailbox string, criteria *imap.SearchCriteria) ([]*imap.Message, error) {
	imapClient, err := connectToMailbox(dialer, account, mailbox, true)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mailbox: %w", err)
	}
	defer imapClient.Logout()

	if err := logServerCapabilities(imapClient); err != nil {
		return nil, fmt.Errorf("failed to log server capabilities: %w", err)
	}

	uids, err := findMessageUIDs(imapClient, criteria)
	if err != nil {
		return nil, err
	}

	if len(uids) == 0 {
		return []*imap.Message{}, nil
	}

	messages, err := fetchMessagesByUID(imapClient, uids)
	if err != nil {
		return nil, err
	}

	// Ordering is applied by the caller via SortMessages (the find command
	// exposes --sort); leave the fetched order untouched here.
	return messages, nil
}

// logServerCapabilities retrieves and logs server capabilities
func logServerCapabilities(imapClient IMAPClient) error {
	caps, err := imapClient.Capability()
	if err != nil {
		return fmt.Errorf("failed to get capabilities: %w", err)
	}

	log.Debug().Msgf("Server capabilities:")
	for serverCap := range caps {
		log.Debug().Msgf("- %s", serverCap)
	}
	return nil
}

// findMessageUIDs searches for messages matching the criteria and returns their UIDs
func findMessageUIDs(client IMAPClient, criteria *imap.SearchCriteria) ([]uint32, error) {
	uids, err := client.UidSearch(criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}
	return uids, nil
}

// fetchMessagesByUID fetches full message data for the given UIDs
func fetchMessagesByUID(client IMAPClient, uids []uint32) ([]*imap.Message, error) {
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)

	messages := make(chan *imap.Message)
	done := make(chan error, 1)

	items := getFetchItems()

	go func() {
		done <- client.UidFetch(seqSet, items, messages)
	}()

	// go-imap (and some servers, e.g. Dovecot with CONDSTORE/QRESYNC) can
	// deliver more than one FETCH response per message. Collapse repeats by UID,
	// merging any fields a later response fills in, so callers see each message
	// exactly once without losing data.
	seen := make(map[uint32]*imap.Message, len(uids))
	var result []*imap.Message
	for msg := range messages {
		if existing, ok := seen[msg.Uid]; ok {
			mergeMessage(existing, msg)
			continue
		}
		seen[msg.Uid] = msg
		result = append(result, msg)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	return result, nil
}

// mergeMessage fills zero-valued fields of dst from src, combining multiple
// FETCH responses for the same message.
func mergeMessage(dst, src *imap.Message) {
	if dst.Envelope == nil {
		dst.Envelope = src.Envelope
	}
	if dst.InternalDate.IsZero() {
		dst.InternalDate = src.InternalDate
	}
	if dst.Size == 0 {
		dst.Size = src.Size
	}
	if len(dst.Flags) == 0 {
		dst.Flags = src.Flags
	}
}

// getFetchItems returns the list of items to fetch for each message
func getFetchItems() []imap.FetchItem {
	return []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchInternalDate,
		imap.FetchRFC822Size,
		imap.FetchUid,
	}
}

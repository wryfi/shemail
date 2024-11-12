package imaputils

import (
	"encoding/json"
	"fmt"
	"github.com/emersion/go-imap"
	"time"
)

// SearchOptions represents the optional search parameters
type SearchOptions struct {
	To        *string    // Optional To address
	From      *string    // Optional From address
	Subject   *string    // Optional Subject
	StartDate *time.Time // Optional start date
	EndDate   *time.Time // Optional end date
	Seen      *bool      // Optional seen flag
	Unseen    *bool      // Optional unseen flag
}

// Serialize serializes SearchOptions to json
func (opts SearchOptions) Serialize() string {
	jsonBytes, _ := json.MarshalIndent(opts, "", "  ")
	return string(jsonBytes)
}

// SearchMessages performs a search for messages in the specified mailbox using given criteria
func SearchMessages(dialer IMAPDialer, account Account, mailbox string, criteria *imap.SearchCriteria) ([]*imap.Message, error) {
	imapClient, err := connectToMailbox(account, mailbox, true, dialer)
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

	sortMessagesByDate(messages)
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

	var result []*imap.Message
	for msg := range messages {
		result = append(result, msg)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	return result, nil
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

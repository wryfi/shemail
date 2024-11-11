package imaputils

import (
	"encoding/json"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
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
	jsonBytes, err := json.MarshalIndent(opts, "", "  ")
	if err != nil {
		log.Fatal().Msg("failed to serialize search options to JSON")
	}
	return string(jsonBytes)
}

// SearchMessages performs a search for messages in the specified mailbox using given criteria
func SearchMessages(account Account, mailbox string, criteria *imap.SearchCriteria) ([]*imap.Message, error) {
	imapClient, err := connectAndSelectMailbox(account, mailbox)
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
func logServerCapabilities(imapClient *client.Client) error {
	caps, err := imapClient.Capability()
	if err != nil {
		return fmt.Errorf("failed to get capabilities: %w", err)
	}

	log.Debug().Msgf("Server capabilities:")
	for cap := range caps {
		log.Debug().Msgf("- %s", cap)
	}
	return nil
}

// findMessageUIDs searches for messages matching the criteria and returns their UIDs
func findMessageUIDs(client *client.Client, criteria *imap.SearchCriteria) ([]uint32, error) {
	uids, err := client.UidSearch(criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}
	return uids, nil
}

// fetchMessagesByUID fetches full message data for the given UIDs
func fetchMessagesByUID(client *client.Client, uids []uint32) ([]*imap.Message, error) {
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

// batchMessages splits a slice of messages into batches of the specified size
func batchMessages(messages []*imap.Message, batchSize int) [][]*imap.Message {
	if batchSize <= 0 {
		batchSize = 100 // Default batch size
	}

	var batches [][]*imap.Message
	for i := 0; i < len(messages); i += batchSize {
		end := i + batchSize
		if end > len(messages) {
			end = len(messages)
		}
		batches = append(batches, messages[i:end])
	}
	return batches
}

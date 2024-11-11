package imap

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
		return nil, err
	}
	defer imapClient.Logout()

	if err := logServerCapabilities(imapClient); err != nil {
		return nil, err
	}

	seqNums, err := searchMessageSequences(imapClient, criteria)
	if err != nil {
		return nil, err
	}

	if len(seqNums) == 0 {
		return []*imap.Message{}, nil
	}

	messages, err := fetchMessages(imapClient, seqNums)
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

// searchMessageSequences performs the IMAP search and returns sequence numbers
func searchMessageSequences(imapClient *client.Client, criteria *imap.SearchCriteria) ([]uint32, error) {
	seqNums, err := imapClient.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	return seqNums, nil
}

// fetchMessages retrieves the full message data for the given sequence numbers
func fetchMessages(imapClient *client.Client, seqNums []uint32) ([]*imap.Message, error) {
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(seqNums...)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	var results []*imap.Message

	go func() {
		done <- imapClient.Fetch(seqSet, []imap.FetchItem{
			imap.FetchAll,
		}, messages)
	}()

	for msg := range messages {
		results = append(results, msg)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	return results, nil
}

package imaputils

import (
	"fmt"
	"github.com/emersion/go-imap"
)

// MessageFields represents the fields to fetch from IMAP messages
type MessageFields struct {
	Envelope  bool
	Body      bool
	Flags     bool
	Headers   []string // Specific headers to fetch
	BodyPeek  bool     // Whether to mark messages as read when fetching
	Structure bool     // Message structure/MIME parts
	Size      bool     // Message size
	UID       bool     // Message UID
	All       bool     // Fetch all fields (overrides other options)
}

// DefaultMessageFields returns MessageFields with commonly used defaults
func DefaultMessageFields() MessageFields {
	return MessageFields{
		Envelope: true,
		Headers:  []string{"From", "Subject", "Date"},
		BodyPeek: true,
	}
}

// FetchMessages fetches a list of messages from the specified mailbox with customizable field selection.
func FetchMessages(account Account, mailbox string, fields MessageFields) ([]*imap.Message, error) {
	imapClient, err := getImapClient(account)
	if err != nil {
		return nil, fmt.Errorf("error getting imap client: %w", err)
	}
	defer imapClient.Logout()

	// Select mailbox
	mbox, err := imapClient.Select(mailbox, true)
	if err != nil {
		return nil, fmt.Errorf("failed to select mailbox: %w", err)
	}

	// If mailbox is empty, return early
	if mbox.Messages == 0 {
		return []*imap.Message{}, nil
	}

	// Build fetch items based on requested fields
	items := buildFetchItems(fields)

	// Pre-allocate slice with known capacity
	fetchedMessages := make([]*imap.Message, 0, mbox.Messages)

	// Use batch processing for large mailboxes
	const batchSize = 1000
	for i := uint32(1); i <= mbox.Messages; i += batchSize {
		seqset := new(imap.SeqSet)
		end := i + batchSize - 1
		if end > mbox.Messages {
			end = mbox.Messages
		}
		seqset.AddRange(i, end)

		messages := make(chan *imap.Message, batchSize)
		done := make(chan error, 1)

		go func() {
			done <- imapClient.Fetch(seqset, items, messages)
		}()

		for msg := range messages {
			fetchedMessages = append(fetchedMessages, msg)
		}

		if err := <-done; err != nil {
			return nil, fmt.Errorf("failed to fetch messages batch %d-%d: %w", i, end, err)
		}
	}

	return fetchedMessages, nil
}

// buildFetchItems converts MessageFields into IMAP fetch items
func buildFetchItems(fields MessageFields) []imap.FetchItem {
	if fields.All {
		return []imap.FetchItem{imap.FetchAll}
	}

	items := make([]imap.FetchItem, 0)

	if fields.Envelope {
		items = append(items, imap.FetchEnvelope)
	}

	if fields.Flags {
		items = append(items, imap.FetchFlags)
	}

	if fields.Size {
		items = append(items, imap.FetchRFC822Size)
	}

	if fields.UID {
		items = append(items, imap.FetchUid)
	}

	if fields.Structure {
		items = append(items, imap.FetchBodyStructure)
	}

	if len(fields.Headers) > 0 {
		section := &imap.BodySectionName{
			BodyPartName: imap.BodyPartName{
				Specifier: "HEADER.FIELDS",
				Fields:    fields.Headers,
			},
		}
		items = append(items, section.FetchItem())
	}

	if fields.Body {
		bodySection := &imap.BodySectionName{}
		items = append(items, bodySection.FetchItem())
	}

	return items
}

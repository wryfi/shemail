package imaputils

import (
	"github.com/emersion/go-imap"
)

// FindDuplicates returns the duplicate messages in the slice: every message
// whose Message-ID was already seen earlier in the slice. The first message of
// each Message-ID is kept (not returned), so the caller controls which copy
// survives by ordering the input (e.g. oldest first to keep the original).
// Messages without a Message-ID are never considered duplicates.
func FindDuplicates(messages []*imap.Message) []*imap.Message {
	seen := make(map[string]struct{}, len(messages))
	var duplicates []*imap.Message

	for _, message := range messages {
		if message.Envelope == nil || message.Envelope.MessageId == "" {
			continue
		}

		messageID := message.Envelope.MessageId
		if _, ok := seen[messageID]; ok {
			duplicates = append(duplicates, message)
		} else {
			seen[messageID] = struct{}{}
		}
	}

	return duplicates
}

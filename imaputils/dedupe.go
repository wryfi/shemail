package imaputils

import (
	"github.com/emersion/go-imap"
)

// FindDuplicates returns the duplicate messages in the slice: every message
// whose Message-ID was already seen earlier in the slice. The first message of
// each Message-ID is kept (not returned), so the caller controls which copy
// survives by ordering the input (e.g. oldest first to keep the original).
// Messages without a Message-ID are never considered duplicates.
//
// The input is assumed to contain each UID at most once; fetchMessagesByUID
// guarantees this by collapsing any repeated FETCH responses, so a message can
// never be flagged as a duplicate of itself.
func FindDuplicates(messages []*imap.Message) []*imap.Message {
	// originals maps a Message-ID to the first (kept) message bearing it, so we
	// can log exactly which message each duplicate was matched against.
	originals := make(map[string]*imap.Message, len(messages))
	var duplicates []*imap.Message

	for _, message := range messages {
		if message.Envelope == nil || message.Envelope.MessageId == "" {
			continue
		}

		messageID := message.Envelope.MessageId
		if original, ok := originals[messageID]; ok {
			log.Debug().Msgf(
				"duplicate Message-ID %q: uid=%d subj=%q matches kept uid=%d subj=%q",
				messageID,
				message.Uid, message.Envelope.Subject,
				original.Uid, original.Envelope.Subject,
			)
			duplicates = append(duplicates, message)
		} else {
			originals[messageID] = message
		}
	}

	return duplicates
}

package imaputils

import (
	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFindDuplicates(t *testing.T) {
	msg := func(uid uint32, messageID string) *imap.Message {
		return &imap.Message{Uid: uid, Envelope: &imap.Envelope{MessageId: messageID}}
	}
	uidsOf := func(messages []*imap.Message) []uint32 {
		uids := make([]uint32, len(messages))
		for index, message := range messages {
			uids[index] = message.Uid
		}
		return uids
	}

	t.Run("no duplicates", func(t *testing.T) {
		messages := []*imap.Message{msg(1, "a"), msg(2, "b"), msg(3, "c")}
		assert.Empty(t, FindDuplicates(messages))
	})

	t.Run("keeps the first occurrence and returns the rest", func(t *testing.T) {
		messages := []*imap.Message{
			msg(1, "a"), msg(2, "b"), msg(3, "a"), msg(4, "a"), msg(5, "b"),
		}
		// a: keep 1, duplicates 3 and 4; b: keep 2, duplicate 5.
		assert.Equal(t, []uint32{3, 4, 5}, uidsOf(FindDuplicates(messages)))
	})

	t.Run("messages without a Message-ID are never duplicates", func(t *testing.T) {
		messages := []*imap.Message{
			msg(1, ""),
			msg(2, ""),
			{Uid: 3, Envelope: nil},
		}
		assert.Empty(t, FindDuplicates(messages))
	})
}

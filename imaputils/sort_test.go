package imaputils

import (
	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestParseSortField(t *testing.T) {
	for _, name := range []string{"date", "DATE", "subject", "from", "to", "size", "unread"} {
		field, err := ParseSortField(name)
		assert.NoError(t, err)
		assert.NotEmpty(t, field)
	}
	if _, err := ParseSortField("bogus"); err == nil {
		t.Fatal("expected an error for an unknown sort field")
	}
}

func TestSortMessages(t *testing.T) {
	addr := func(local string) []*imap.Address {
		return []*imap.Address{{MailboxName: local, HostName: "example.com"}}
	}
	// m1: oldest, smallest, read,   subject "banana", from zoe,   to bob
	m1 := &imap.Message{
		Uid: 1, Size: 100, Flags: []string{imap.SeenFlag},
		InternalDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Envelope:     &imap.Envelope{Subject: "banana", From: addr("zoe"), To: addr("bob")},
	}
	// m2: middle,  largest,  unread, subject "apple",  from alice, to dave
	m2 := &imap.Message{
		Uid: 2, Size: 300, Flags: nil,
		InternalDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		Envelope:     &imap.Envelope{Subject: "apple", From: addr("alice"), To: addr("dave")},
	}
	// m3: newest,  middle,   read,   subject "cherry", from mike,  to carol
	m3 := &imap.Message{
		Uid: 3, Size: 200, Flags: []string{imap.SeenFlag},
		InternalDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Envelope:     &imap.Envelope{Subject: "cherry", From: addr("mike"), To: addr("carol")},
	}

	uidsOf := func(messages []*imap.Message) []uint32 {
		uids := make([]uint32, len(messages))
		for index, message := range messages {
			uids[index] = message.Uid
		}
		return uids
	}
	fresh := func() []*imap.Message { return []*imap.Message{m1, m2, m3} }

	cases := []struct {
		name    string
		field   SortField
		reverse bool
		want    []uint32
	}{
		{"date default newest first", SortDate, false, []uint32{3, 2, 1}},
		{"date reverse oldest first", SortDate, true, []uint32{1, 2, 3}},
		{"subject ascending", SortSubject, false, []uint32{2, 1, 3}},       // apple, banana, cherry
		{"size largest first", SortSize, false, []uint32{2, 3, 1}},         // 300, 200, 100
		{"from ascending", SortFrom, false, []uint32{2, 3, 1}},             // alice, mike, zoe
		{"to ascending", SortTo, false, []uint32{1, 3, 2}},                 // bob, carol, dave
		{"unread first then newest", SortUnread, false, []uint32{2, 3, 1}}, // m2 unread; m3,m1 by date
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			messages := fresh()
			SortMessages(messages, tt.field, tt.reverse)
			assert.Equal(t, tt.want, uidsOf(messages))
		})
	}
}

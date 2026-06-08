package imaputils

import (
	"fmt"
	"github.com/emersion/go-imap"
	"sort"
	"strings"
)

// SortField identifies the message attribute to sort by.
type SortField string

const (
	SortDate    SortField = "date"
	SortSubject SortField = "subject"
	SortFrom    SortField = "from"
	SortTo      SortField = "to"
	SortSize    SortField = "size"
	SortUnread  SortField = "unread"
)

// ParseSortField validates and normalizes a sort field name.
func ParseSortField(name string) (SortField, error) {
	field := SortField(strings.ToLower(name))
	switch field {
	case SortDate, SortSubject, SortFrom, SortTo, SortSize, SortUnread:
		return field, nil
	default:
		return "", fmt.Errorf("unknown sort field %q (valid: date, subject, from, to, size, unread)", name)
	}
}

// SortMessages sorts messages in place by the given field. Each field has a
// sensible default direction — date newest-first, size largest-first, text
// fields alphabetical, unread before read — and reverse inverts it. Ties fall
// back to a stable reverse-chronological order.
func SortMessages(messages []*imap.Message, field SortField, reverse bool) {
	// Baseline chronological order (newest first) so that ties in any other
	// field break sensibly and stably.
	sortMessagesByDate(messages)

	less := lessFuncFor(field)
	sort.SliceStable(messages, func(i, j int) bool {
		if reverse {
			return less(messages[j], messages[i])
		}
		return less(messages[i], messages[j])
	})
}

func lessFuncFor(field SortField) func(a, b *imap.Message) bool {
	switch field {
	case SortSubject:
		return func(a, b *imap.Message) bool { return subjectSortKey(a) < subjectSortKey(b) }
	case SortFrom:
		return func(a, b *imap.Message) bool { return fromSortKey(a) < fromSortKey(b) }
	case SortTo:
		return func(a, b *imap.Message) bool { return toSortKey(a) < toSortKey(b) }
	case SortSize:
		return func(a, b *imap.Message) bool { return a.Size > b.Size }
	case SortUnread:
		return func(a, b *imap.Message) bool { return messageUnread(a) && !messageUnread(b) }
	default: // SortDate
		return func(a, b *imap.Message) bool { return a.InternalDate.After(b.InternalDate) }
	}
}

func subjectSortKey(message *imap.Message) string {
	if message.Envelope == nil {
		return ""
	}
	return strings.ToLower(message.Envelope.Subject)
}

func fromSortKey(message *imap.Message) string {
	if message.Envelope == nil || len(message.Envelope.From) == 0 {
		return ""
	}
	return strings.ToLower(FormatAddress(message.Envelope.From[0]))
}

func toSortKey(message *imap.Message) string {
	if message.Envelope == nil || len(message.Envelope.To) == 0 {
		return ""
	}
	return strings.ToLower(FormatAddress(message.Envelope.To[0]))
}

func messageUnread(message *imap.Message) bool {
	for _, flag := range message.Flags {
		if flag == imap.SeenFlag {
			return false
		}
	}
	return true
}

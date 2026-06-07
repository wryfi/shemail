package imaputils

import (
	"fmt"
	"github.com/emersion/go-imap"
	"sort"
	"strings"
)

// FormatAddress formats an IMAP address into a human-readable string. It
// returns an empty string when the address has neither a mailbox nor a host,
// so callers can distinguish "no usable address" from a real one (rather than
// getting a bare "@").
func FormatAddress(address *imap.Address) string {
	if address == nil {
		return ""
	}
	switch {
	case address.MailboxName == "" && address.HostName == "":
		return ""
	case address.HostName == "":
		return address.MailboxName
	case address.MailboxName == "":
		return address.HostName
	default:
		return fmt.Sprintf("%s@%s", address.MailboxName, address.HostName)
	}
}

// FormatAddresses formats a slice of IMAP addresses into a comma-separated string.
func FormatAddresses(addresses []*imap.Address) []string {
	formatted := []string{}
	for _, addr := range addresses {
		formatted = append(formatted, FormatAddress(addr))
	}
	return formatted
}

// FormatAddressesCSV formats a slice of IMAP addresses into a comma-separated string.
func FormatAddressesCSV(addresses []*imap.Address) string {
	formatted := FormatAddresses(addresses)
	count := len(formatted)
	if count > 1 {
		joined := strings.Join(formatted[0:1], ", ")
		return fmt.Sprintf("%s (+%d)", joined, count-1)
	}
	return strings.Join(formatted, ", ")
}

// sortMessagesByDate sorts messages in reverse chronological order
func sortMessagesByDate(messages []*imap.Message) {
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].InternalDate.After(messages[j].InternalDate)
	})
}

// createSeqSet should work with UIDs, not sequence numbers
func createSeqSet(messages []*imap.Message) *imap.SeqSet {
	seqSet := new(imap.SeqSet)
	for _, msg := range messages {
		seqSet.AddNum(msg.Uid) // Use UID instead of sequence number
	}
	return seqSet
}

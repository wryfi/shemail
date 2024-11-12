package imaputils

import (
	"fmt"
	"github.com/emersion/go-imap"
	"sort"
	"strings"
)

// FormatAddress formats an IMAP address into a human-readable string.
func FormatAddress(address *imap.Address) string {
	//var name, mailbox, host string
	var mailbox, host string
	//if address.PersonalName != "" {
	//	name = address.PersonalName
	//}

	if address.MailboxName != "" && address.HostName != "" {
		mailbox = address.MailboxName
		host = address.HostName
	}

	//if name != "" {
	//	return fmt.Sprintf("%s <%s@%s>", name, mailbox, host)
	//}
	return fmt.Sprintf("%s@%s", mailbox, host)
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

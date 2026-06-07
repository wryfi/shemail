package imaputils

import (
	"github.com/emersion/go-imap"
	"reflect"
	"testing"
	"time"
)

func TestFormatAddress(t *testing.T) {
	address := &imap.Address{
		MailboxName: "mailbox",
		HostName:    "example.com",
	}
	expected := "mailbox@example.com"
	result := FormatAddress(address)
	if result != expected {
		t.Errorf("Expected %s, but got %s", expected, result)
	}
}

func TestFormatAddressEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		address  *imap.Address
		expected string
	}{
		{"nil address", nil, ""},
		{"empty address", &imap.Address{}, ""},
		{"mailbox only", &imap.Address{MailboxName: "mailbox"}, "mailbox"},
		{"host only", &imap.Address{HostName: "example.com"}, "example.com"},
		{"full", &imap.Address{MailboxName: "mailbox", HostName: "example.com"}, "mailbox@example.com"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if result := FormatAddress(test.address); result != test.expected {
				t.Errorf("Expected %q, but got %q", test.expected, result)
			}
		})
	}
}

func TestFormatAddresses(t *testing.T) {
	addresses := []*imap.Address{
		{MailboxName: "mailbox1", HostName: "example.com"},
		{MailboxName: "mailbox2", HostName: "example.org"},
	}
	expected := []string{"mailbox1@example.com", "mailbox2@example.org"}
	result := FormatAddresses(addresses)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestFormatAddressesCSV(t *testing.T) {
	addresses := []*imap.Address{
		{MailboxName: "mailbox1", HostName: "example.com"},
		{MailboxName: "mailbox2", HostName: "example.org"},
	}
	expected := "mailbox1@example.com (+1)"
	result := FormatAddressesCSV(addresses)
	if result != expected {
		t.Errorf("Expected %s, but got %s", expected, result)
	}
}

func TestSortMessagesByDate(t *testing.T) {
	msg1 := &imap.Message{InternalDate: time.Now().Add(-10 * time.Hour)}
	msg2 := &imap.Message{InternalDate: time.Now()}
	messages := []*imap.Message{msg1, msg2}
	sortMessagesByDate(messages)
	if messages[0] != msg2 && messages[1] != msg1 {
		t.Errorf("Messages are not sorted in reverse chronological order")
	}
}

func TestCreateSeqSet(t *testing.T) {
	msg1 := &imap.Message{Uid: 100}
	msg2 := &imap.Message{Uid: 101}
	messages := []*imap.Message{msg1, msg2}
	seqSet := createSeqSet(messages)
	expected := &imap.SeqSet{}
	expected.AddNum(100)
	expected.AddNum(101)
	if !reflect.DeepEqual(seqSet, expected) {
		t.Errorf("Expected %v, but got %v", expected, seqSet)
	}
}

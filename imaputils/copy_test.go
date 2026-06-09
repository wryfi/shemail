package imaputils

import (
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestCopyMessages(t *testing.T) {
	t.Run("empty message set is a no-op", func(t *testing.T) {
		client := &MockIMAPClientMove{}
		dialer := &MockIMAPDialerMove{}

		err := CopyMessages(dialer, Account{Server: "test.example.com"}, nil, "INBOX", "Archive")

		assert.NoError(t, err)
		client.AssertExpectations(t)
		dialer.AssertExpectations(t)
	})

	t.Run("copies to an existing destination", func(t *testing.T) {
		client := &MockIMAPClientMove{}
		dialer := &MockIMAPDialerMove{}

		dialer.On("Dial", mock.Anything).Return(client, nil)
		client.On("Login", mock.Anything, mock.Anything).Return(nil)
		client.On("Logout").Return(nil)

		// EnsureFolder: hierarchy delimiter discovery + existence check (exists).
		client.On("List", "", "", mock.Anything).Return(
			func(ch chan *imap.MailboxInfo) { ch <- &imap.MailboxInfo{Delimiter: "/"} }, nil,
		)
		client.On("List", "", "Archive", mock.Anything).Return(
			func(ch chan *imap.MailboxInfo) { ch <- &imap.MailboxInfo{Name: "Archive"} }, nil,
		)

		// Source is selected read-only (COPY does not modify the source) and the
		// messages are copied by UID.
		client.On("Select", "INBOX", true).Return(&imap.MailboxStatus{}, nil)
		client.On("UidCopy", mock.Anything, "Archive").Return(nil)

		messages := []*imap.Message{{Uid: 1}, {Uid: 2}}
		err := CopyMessages(dialer, Account{Server: "test.example.com"}, messages, "INBOX", "Archive")

		assert.NoError(t, err)
		client.AssertExpectations(t)
	})

	t.Run("propagates copy errors", func(t *testing.T) {
		client := &MockIMAPClientMove{}
		dialer := &MockIMAPDialerMove{}

		dialer.On("Dial", mock.Anything).Return(client, nil)
		client.On("Login", mock.Anything, mock.Anything).Return(nil)
		client.On("Logout").Return(nil)
		client.On("List", "", "", mock.Anything).Return(
			func(ch chan *imap.MailboxInfo) { ch <- &imap.MailboxInfo{Delimiter: "/"} }, nil,
		)
		client.On("List", "", "Archive", mock.Anything).Return(
			func(ch chan *imap.MailboxInfo) { ch <- &imap.MailboxInfo{Name: "Archive"} }, nil,
		)
		client.On("Select", "INBOX", true).Return(&imap.MailboxStatus{}, nil)
		client.On("UidCopy", mock.Anything, "Archive").Return(fmt.Errorf("copy failed"))

		err := CopyMessages(dialer, Account{Server: "test.example.com"}, []*imap.Message{{Uid: 1}}, "INBOX", "Archive")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "copy failed")
	})
}

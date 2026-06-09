package imaputils

import (
	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestFindTrashFolder(t *testing.T) {
	listing := func(names ...string) func(chan *imap.MailboxInfo) {
		return func(ch chan *imap.MailboxInfo) {
			for _, name := range names {
				ch <- &imap.MailboxInfo{Name: name}
			}
		}
	}

	cases := []struct {
		name    string
		folders []string
		want    string
	}{
		{"matches Trash", []string{"INBOX", "Trash", "Sent"}, "Trash"},
		{"matches gmail trash", []string{"INBOX", "[Gmail]/Trash"}, "[Gmail]/Trash"},
		{"falls back to default when none match", []string{"INBOX", "Sent"}, "Deleted Items"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockIMAPClientMove{}
			dialer := &MockIMAPDialerMove{}
			dialer.On("Dial", mock.Anything).Return(client, nil)
			client.On("Login", mock.Anything, mock.Anything).Return(nil)
			client.On("Logout").Return(nil)
			client.On("List", "", "*", mock.Anything).Return(listing(tt.folders...), nil)

			folder, err := FindTrashFolder(dialer, Account{Server: "test.example.com"})
			assert.NoError(t, err)
			assert.Equal(t, tt.want, folder)
		})
	}
}

func TestEmptyFolder(t *testing.T) {
	t.Run("permanently deletes every message in the folder", func(t *testing.T) {
		client := &MockIMAPClientMove{}
		dialer := &MockIMAPDialerMove{}

		dialer.On("Dial", mock.Anything).Return(client, nil)
		client.On("Login", mock.Anything, mock.Anything).Return(nil)
		client.On("Logout").Return(nil)
		client.On("Select", "Trash", false).Return(&imap.MailboxStatus{}, nil)
		client.On("UidSearch", mock.Anything).Return([]uint32{10, 11, 12}, nil)
		client.On("UidStore",
			mock.Anything,
			imap.FormatFlagsOp(imap.AddFlags, true),
			[]interface{}{imap.DeletedFlag},
			(chan *imap.Message)(nil),
		).Return(nil, nil)
		client.On("Expunge", (chan uint32)(nil)).Return(nil)

		count, err := EmptyFolder(dialer, Account{Server: "test.example.com"}, "Trash")
		assert.NoError(t, err)
		assert.Equal(t, 3, count)
		client.AssertExpectations(t)
	})

	t.Run("empty folder does not store or expunge", func(t *testing.T) {
		client := &MockIMAPClientMove{}
		dialer := &MockIMAPDialerMove{}

		dialer.On("Dial", mock.Anything).Return(client, nil)
		client.On("Login", mock.Anything, mock.Anything).Return(nil)
		client.On("Logout").Return(nil)
		client.On("Select", "Trash", false).Return(&imap.MailboxStatus{}, nil)
		client.On("UidSearch", mock.Anything).Return([]uint32{}, nil)

		count, err := EmptyFolder(dialer, Account{Server: "test.example.com"}, "Trash")
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
		client.AssertExpectations(t)
	})
}

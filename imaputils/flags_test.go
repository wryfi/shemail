package imaputils

import (
	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestMarkMessages(t *testing.T) {
	tests := []struct {
		name        string
		seen        bool
		messages    []*imap.Message
		expectStore bool
		expectedOp  imap.StoreItem
	}{
		{
			name:        "mark read adds Seen",
			seen:        true,
			messages:    []*imap.Message{{Uid: 1}, {Uid: 2}},
			expectStore: true,
			expectedOp:  imap.FormatFlagsOp(imap.AddFlags, true),
		},
		{
			name:        "mark unread removes Seen",
			seen:        false,
			messages:    []*imap.Message{{Uid: 1}},
			expectStore: true,
			expectedOp:  imap.FormatFlagsOp(imap.RemoveFlags, true),
		},
		{
			name:        "empty message set is a no-op",
			seen:        true,
			messages:    nil,
			expectStore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockIMAPClientMove{}
			dialer := &MockIMAPDialerMove{}

			if tt.expectStore {
				dialer.On("Dial", mock.Anything).Return(client, nil)
				client.On("Login", mock.Anything, mock.Anything).Return(nil)
				client.On("Select", "INBOX", false).Return(&imap.MailboxStatus{}, nil)
				client.On("UidStore",
					mock.Anything,
					tt.expectedOp,
					[]interface{}{imap.SeenFlag},
					(chan *imap.Message)(nil),
				).Return(nil, nil)
				client.On("Logout").Return(nil)
			}

			err := MarkMessages(dialer, Account{Server: "test.example.com"}, tt.messages, "INBOX", tt.seen)
			assert.NoError(t, err)
			client.AssertExpectations(t)
		})
	}
}

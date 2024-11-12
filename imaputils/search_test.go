package imaputils

import (
	"crypto/tls"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

type MockIMAPClientSearch struct {
	mock.Mock
}

func (m *MockIMAPClientSearch) Capability() (map[string]bool, error) {
	args := m.Called()
	return args.Get(0).(map[string]bool), args.Error(1)
}

func (m *MockIMAPClientSearch) UidSearch(criteria *imap.SearchCriteria) ([]uint32, error) {
	args := m.Called(criteria)
	return args.Get(0).([]uint32), args.Error(1)
}

func (m *MockIMAPClientSearch) UidFetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	args := m.Called(seqset, items, ch)
	// Simulate async message delivery
	if msgs, ok := args.Get(0).([]*imap.Message); ok && msgs != nil {
		go func() {
			for _, msg := range msgs {
				ch <- msg
			}
			close(ch)
		}()
	}
	return args.Error(1)
}

func (m *MockIMAPClientSearch) Logout() error {
	args := m.Called()
	return args.Error(0)
}

// Other interface methods...
func (m *MockIMAPClientSearch) Create(name string) error     { return nil }
func (m *MockIMAPClientSearch) Expunge(ch chan uint32) error { return nil }
func (m *MockIMAPClientSearch) Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	return nil
}
func (m *MockIMAPClientSearch) GetClient() *client.Client { return nil }
func (m *MockIMAPClientSearch) List(ref string, name string, ch chan *imap.MailboxInfo) error {
	return nil
}
func (m *MockIMAPClientSearch) Login(username string, password string) error { return nil }
func (m *MockIMAPClientSearch) Select(name string, readOnly bool) (*imap.MailboxStatus, error) {
	return nil, nil
}
func (m *MockIMAPClientSearch) UidCopy(seqSet *imap.SeqSet, dest string) error { return nil }
func (m *MockIMAPClientSearch) UidDelete(seqSet *imap.SeqSet) error            { return nil }
func (m *MockIMAPClientSearch) UidFetchMetadata(seqSet *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	return nil
}
func (m *MockIMAPClientSearch) UidMove(seqSet *imap.SeqSet, mailbox string) error { return nil }
func (m *MockIMAPClientSearch) UidStore(seqSet *imap.SeqSet, item imap.StoreItem, flags []interface{}, ch chan *imap.Message) error {
	return nil
}

type MockIMAPDialerSearch struct {
	mock.Mock
}

func (m *MockIMAPDialerSearch) Dial(address string) (IMAPClient, error) {
	args := m.Called(address)
	return args.Get(0).(IMAPClient), args.Error(1)
}

func (m *MockIMAPDialerSearch) DialTLS(address string, config *tls.Config) (IMAPClient, error) {
	args := m.Called(address, config)
	return args.Get(0).(IMAPClient), args.Error(1)
}

func TestSearchMessages(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockIMAPClientSearch, *MockIMAPDialerSearch)
		expectedUIDs   []uint32
		expectedMsgs   []*imap.Message
		expectedError  string
		searchCriteria *imap.SearchCriteria
	}{
		{
			name: "successful search with results",
			setupMocks: func(client *MockIMAPClientSearch, dialer *MockIMAPDialerSearch) {
				uids := []uint32{1, 2, 3}
				messages := []*imap.Message{
					{Uid: 1, Envelope: &imap.Envelope{Subject: "Test 1"}},
					{Uid: 2, Envelope: &imap.Envelope{Subject: "Test 2"}},
					{Uid: 3, Envelope: &imap.Envelope{Subject: "Test 3"}},
				}

				capabilities := map[string]bool{"IMAP4rev1": true}
				client.On("Capability").Return(capabilities, nil)
				client.On("UidSearch", mock.Anything).Return(uids, nil)
				client.On("UidFetch", mock.Anything, mock.Anything, mock.Anything).Return(messages, nil)
				client.On("Logout").Return(nil)

				dialer.On("Dial", mock.Anything).Return(client, nil)
			},
			expectedUIDs:   []uint32{1, 2, 3},
			expectedMsgs:   []*imap.Message{{Uid: 1}, {Uid: 2}, {Uid: 3}},
			searchCriteria: &imap.SearchCriteria{},
		},
		{
			name: "no messages found",
			setupMocks: func(client *MockIMAPClientSearch, dialer *MockIMAPDialerSearch) {
				capabilities := map[string]bool{"IMAP4rev1": true}
				client.On("Capability").Return(capabilities, nil)
				client.On("UidSearch", mock.Anything).Return([]uint32{}, nil)
				client.On("Logout").Return(nil)

				dialer.On("Dial", mock.Anything).Return(client, nil)
			},
			expectedUIDs:   []uint32{},
			expectedMsgs:   []*imap.Message{},
			searchCriteria: &imap.SearchCriteria{},
		},
		{
			name: "search error",
			setupMocks: func(client *MockIMAPClientSearch, dialer *MockIMAPDialerSearch) {
				capabilities := map[string]bool{"IMAP4rev1": true}
				client.On("Capability").Return(capabilities, nil)
				client.On("UidSearch", mock.Anything).Return([]uint32{}, fmt.Errorf("search failed"))
				client.On("Logout").Return(nil)

				dialer.On("Dial", mock.Anything).Return(client, nil)
			},
			expectedError:  "failed to search messages: search failed",
			searchCriteria: &imap.SearchCriteria{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSearchClient := &MockIMAPClientSearch{}
			dialer := &MockIMAPDialerSearch{}
			tt.setupMocks(mockSearchClient, dialer)

			account := Account{
				User:     "test@example.com",
				Password: "password",
				Server:   "imap.example.com",
				Port:     993,
			}

			messages, err := SearchMessages(dialer, account, "INBOX", tt.searchCriteria)

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Len(t, messages, len(tt.expectedMsgs))
			}

			mockSearchClient.AssertExpectations(t)
			dialer.AssertExpectations(t)
		})
	}
}

func TestSearchOptions_Serialize(t *testing.T) {
	now := time.Now()
	seen := true
	unseen := false

	tests := []struct {
		name     string
		opts     SearchOptions
		expected string
	}{
		{
			name: "full options",
			opts: SearchOptions{
				To:        stringPtr("to@example.com"),
				From:      stringPtr("from@example.com"),
				Subject:   stringPtr("test subject"),
				StartDate: &now,
				EndDate:   &now,
				Seen:      &seen,
				Unseen:    &unseen,
			},
			expected: `{
  "To": "to@example.com",
  "From": "from@example.com",
  "Subject": "test subject",
  "StartDate": "` + now.Format(time.RFC3339Nano) + `",
  "EndDate": "` + now.Format(time.RFC3339Nano) + `",
  "Seen": true,
  "Unseen": false
}`,
		},
		{
			name: "empty options",
			opts: SearchOptions{},
			expected: `{
  "To": null,
  "From": null,
  "Subject": null,
  "StartDate": null,
  "EndDate": null,
  "Seen": null,
  "Unseen": null
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.opts.Serialize()
			assert.JSONEq(t, tt.expected, result)
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

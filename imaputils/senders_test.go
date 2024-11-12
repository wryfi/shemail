package imaputils

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

type MockIMAPClientSenders struct {
	mock.Mock
}

func (m *MockIMAPClientSenders) Capability() (map[string]bool, error) {
	args := m.Called()
	return args.Get(0).(map[string]bool), args.Error(1)
}

func (m *MockIMAPClientSenders) UidSearch(criteria *imap.SearchCriteria) ([]uint32, error) {
	args := m.Called(criteria)
	return args.Get(0).([]uint32), args.Error(1)
}

func (m *MockIMAPClientSenders) UidFetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	args := m.Called(seqset, items, ch)
	// Simulate async message sending if messages were provided
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

func (m *MockIMAPClientSenders) Logout() error {
	args := m.Called()
	return args.Error(0)
}

// Implement other required interface methods with empty returns
func (m *MockIMAPClientSenders) Create(name string) error     { return nil }
func (m *MockIMAPClientSenders) Expunge(ch chan uint32) error { return nil }
func (m *MockIMAPClientSenders) Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	return nil
}
func (m *MockIMAPClientSenders) GetClient() *client.Client { return nil }
func (m *MockIMAPClientSenders) List(ref string, name string, ch chan *imap.MailboxInfo) error {
	return nil
}
func (m *MockIMAPClientSenders) Login(username string, password string) error { return nil }
func (m *MockIMAPClientSenders) Select(name string, readOnly bool) (*imap.MailboxStatus, error) {
	return nil, nil
}
func (m *MockIMAPClientSenders) UidCopy(seqSet *imap.SeqSet, mailbox string) error { return nil }
func (m *MockIMAPClientSenders) UidMove(seqSet *imap.SeqSet, mailbox string) error { return nil }
func (m *MockIMAPClientSenders) UidStore(seqSet *imap.SeqSet, item imap.StoreItem, flags []interface{}, ch chan *imap.Message) error {
	return nil
}

type MockIMAPDialerSenders struct {
	mock.Mock
}

func (m *MockIMAPDialerSenders) Dial(address string) (IMAPClient, error) {
	args := m.Called(address)
	return args.Get(0).(IMAPClient), args.Error(1)
}

func (m *MockIMAPDialerSenders) DialTLS(address string, config *tls.Config) (IMAPClient, error) {
	args := m.Called(address, config)
	return args.Get(0).(IMAPClient), args.Error(1)
}

func TestSearchOptionsSenders_Serialize(t *testing.T) {
	tests := []struct {
		name     string
		opts     SearchOptions
		expected map[string]interface{}
	}{
		{
			name: "all fields populated",
			opts: SearchOptions{
				To:        stringPtr("recipient@example.com"),
				From:      stringPtr("sender@example.com"),
				Subject:   stringPtr("Test Subject"),
				StartDate: timePtr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
				EndDate:   timePtr(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)),
				Seen:      boolPtr(true),
				Unseen:    boolPtr(false),
			},
		},
		{
			name: "empty fields",
			opts: SearchOptions{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized := tt.opts.Serialize()
			var result SearchOptions
			err := json.Unmarshal([]byte(serialized), &result)
			assert.NoError(t, err)

			// Compare original and unmarshaled objects
			assertSearchOptionsEqual(t, tt.opts, result)
		})
	}
}

func TestSearchMessagesSenders(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockIMAPClientSenders)
		expectedMsgs  []*imap.Message
		expectedError string
	}{
		{
			name: "successful search with results",
			setupMocks: func(m *MockIMAPClientSenders) {
				m.On("Capability").Return(map[string]bool{"IMAP4rev1": true}, nil)
				m.On("UidSearch", mock.Anything).Return([]uint32{1, 2}, nil)

				messages := []*imap.Message{
					{Uid: 1, Envelope: &imap.Envelope{Subject: "Test 1"}},
					{Uid: 2, Envelope: &imap.Envelope{Subject: "Test 2"}},
				}
				m.On("UidFetch", mock.Anything, mock.Anything, mock.Anything).Return(messages, nil)
				m.On("Logout").Return(nil)
			},
			expectedMsgs: []*imap.Message{
				{Uid: 1, Envelope: &imap.Envelope{Subject: "Test 1"}},
				{Uid: 2, Envelope: &imap.Envelope{Subject: "Test 2"}},
			},
		},
		{
			name: "search with no results",
			setupMocks: func(m *MockIMAPClientSenders) {
				m.On("Capability").Return(map[string]bool{"IMAP4rev1": true}, nil)
				m.On("UidSearch", mock.Anything).Return([]uint32{}, nil)
				m.On("Logout").Return(nil)
			},
			expectedMsgs: []*imap.Message{},
		},
		{
			name: "capability error",
			setupMocks: func(m *MockIMAPClientSenders) {
				m.On("Capability").Return(map[string]bool{}, fmt.Errorf("capability error"))
				m.On("Logout").Return(nil)
			},
			expectedError: "failed to log server capabilities: failed to get capabilities: capability error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockIMAPClientSenders{}
			if tt.setupMocks != nil {
				tt.setupMocks(mockClient)
			}

			mockDialer := &MockIMAPDialerSenders{}
			mockDialer.On("Dial", mock.Anything).Return(mockClient, nil)

			criteria := &imap.SearchCriteria{}
			messages, err := SearchMessages(mockDialer, Account{}, "INBOX", criteria)

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expectedMsgs), len(messages))
				for i, msg := range messages {
					assert.Equal(t, tt.expectedMsgs[i].Uid, msg.Uid)
					assert.Equal(t, tt.expectedMsgs[i].Envelope.Subject, msg.Envelope.Subject)
				}
			}

			mockClient.AssertExpectations(t)
			mockDialer.AssertExpectations(t)
		})
	}
}

// Helper functions
func timePtr(t time.Time) *time.Time {
	return &t
}

func boolPtr(b bool) *bool {
	return &b
}

func assertSearchOptionsEqual(t *testing.T, expected, actual SearchOptions) {
	assert.Equal(t, expected.To != nil, actual.To != nil)
	if expected.To != nil {
		assert.Equal(t, *expected.To, *actual.To)
	}

	assert.Equal(t, expected.From != nil, actual.From != nil)
	if expected.From != nil {
		assert.Equal(t, *expected.From, *actual.From)
	}

	assert.Equal(t, expected.Subject != nil, actual.Subject != nil)
	if expected.Subject != nil {
		assert.Equal(t, *expected.Subject, *actual.Subject)
	}

	assert.Equal(t, expected.StartDate != nil, actual.StartDate != nil)
	if expected.StartDate != nil {
		assert.True(t, expected.StartDate.Equal(*actual.StartDate))
	}

	assert.Equal(t, expected.EndDate != nil, actual.EndDate != nil)
	if expected.EndDate != nil {
		assert.True(t, expected.EndDate.Equal(*actual.EndDate))
	}

	assert.Equal(t, expected.Seen != nil, actual.Seen != nil)
	if expected.Seen != nil {
		assert.Equal(t, *expected.Seen, *actual.Seen)
	}

	assert.Equal(t, expected.Unseen != nil, actual.Unseen != nil)
	if expected.Unseen != nil {
		assert.Equal(t, *expected.Unseen, *actual.Unseen)
	}
}

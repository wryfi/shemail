package imaputils

import (
	"crypto/tls"
	"errors"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"reflect"
	"testing"
)

type TestIMAPClient struct {
	messages     uint32
	shouldError  bool
	fetchedItems []imap.FetchItem
	client       *client.Client
}

func (m *TestIMAPClient) Capability() (map[string]bool, error) {
	if m.shouldError {
		return nil, errors.New("mock capability error")
	}
	return map[string]bool{"IMAP4rev1": true}, nil
}

func (m *TestIMAPClient) Create(name string) error {
	if m.shouldError {
		return errors.New("mock create error")
	}
	return nil
}

func (m *TestIMAPClient) Expunge(ch chan uint32) error {
	if m.shouldError {
		return errors.New("mock expunge error")
	}
	close(ch)
	return nil
}

// TestIMAPClient.Fetch implementation has a bug. Here's how it should be fixed:
func (m *TestIMAPClient) Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	m.fetchedItems = items
	if m.shouldError {
		return errors.New("mock fetch error")
	}

	// Process each range in the sequence set
	for _, seq := range seqset.Set {
		// For each number in the range
		for i := seq.Start; i <= seq.Stop && i <= m.messages; i++ {
			ch <- &imap.Message{SeqNum: i}
		}
	}
	close(ch)
	return nil
}

func (m *TestIMAPClient) GetClient() *client.Client {
	return m.client
}

func (m *TestIMAPClient) List(ref string, name string, ch chan *imap.MailboxInfo) error {
	if m.shouldError {
		return errors.New("mock list error")
	}
	defer close(ch)
	ch <- &imap.MailboxInfo{Name: "INBOX"}
	return nil
}

func (m *TestIMAPClient) Login(username string, password string) error {
	if m.shouldError {
		return errors.New("mock login error")
	}
	return nil
}

func (m *TestIMAPClient) Logout() error {
	return nil
}

func (m *TestIMAPClient) Select(name string, readOnly bool) (*imap.MailboxStatus, error) {
	if m.shouldError {
		return nil, errors.New("mock select error")
	}
	return &imap.MailboxStatus{Messages: m.messages}, nil
}

func (m *TestIMAPClient) UidCopy(seqset *imap.SeqSet, dest string) error {
	if m.shouldError {
		return errors.New("mock uid copy error")
	}
	return nil
}

func (m *TestIMAPClient) UidFetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	if m.shouldError {
		return errors.New("mock uid fetch error")
	}
	defer close(ch)
	for i := uint32(0); i < m.messages && i < seqset.Set[0].Stop; i++ {
		ch <- &imap.Message{Uid: i + 1}
	}
	return nil
}

func (m *TestIMAPClient) UidMove(seqSet *imap.SeqSet, mailbox string) error {
	if m.shouldError {
		return errors.New("mock uid move error")
	}
	return nil
}

func (m *TestIMAPClient) UidSearch(criteria *imap.SearchCriteria) ([]uint32, error) {
	if m.shouldError {
		return nil, errors.New("mock uid search error")
	}
	return []uint32{1, 2, 3}, nil
}

func (m *TestIMAPClient) UidStore(seqSet *imap.SeqSet, item imap.StoreItem, flags []interface{}, ch chan *imap.Message) error {
	if m.shouldError {
		return errors.New("mock uid store error")
	}
	close(ch)
	return nil
}

type MockDialer struct {
	client *TestIMAPClient
	err    error
}

func (d *MockDialer) Dial(address string) (IMAPClient, error) {
	if d.err != nil {
		return nil, d.err
	}
	return d.client, nil
}

func (d *MockDialer) DialTLS(address string, config *tls.Config) (IMAPClient, error) {
	if d.err != nil {
		return nil, d.err
	}
	return d.client, nil
}

func TestFetchMessages(t *testing.T) {
	tests := []struct {
		name          string
		messages      uint32
		shouldError   bool
		dialerError   error
		expectedLen   int
		expectedError bool
	}{
		{
			name:          "Successful fetch",
			messages:      5,
			shouldError:   false,
			expectedLen:   5,
			expectedError: false,
		},
		{
			name:          "Empty mailbox",
			messages:      0,
			shouldError:   false,
			expectedLen:   0,
			expectedError: false,
		},
		{
			name:          "Dialer error",
			dialerError:   errors.New("dialer error"),
			expectedError: true,
		},
		{
			name:          "Fetch error",
			messages:      10,
			shouldError:   true,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &TestIMAPClient{
				messages:    tt.messages,
				shouldError: tt.shouldError,
			}

			mockDialer := &MockDialer{
				client: client,
				err:    tt.dialerError,
			}

			account := Account{
				Server:   "test.example.com",
				Port:     993,
				User:     "test@example.com",
				Password: "password",
			}

			messages, err := FetchMessages(mockDialer, account, "INBOX", DefaultMessageFields())

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if len(messages) != tt.expectedLen {
					t.Errorf("Expected %d messages, got %d", tt.expectedLen, len(messages))
				}
			}
		})
	}
}

func TestFetchMessagesBatchProcessing(t *testing.T) {
	// Test batch processing with a large number of messages
	myClient := &TestIMAPClient{
		messages:    2500, // More than batchSize (1000)
		shouldError: false,
	}

	mockDialer := &MockDialer{
		client: myClient,
	}

	account := Account{
		Server:   "test.example.com",
		Port:     993,
		User:     "test@example.com",
		Password: "password",
	}

	messages, err := FetchMessages(mockDialer, account, "INBOX", DefaultMessageFields())

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(messages) != 2500 {
		t.Errorf("Expected 2500 messages, got %d", len(messages))
	}
}

func TestDefaultMessageFields(t *testing.T) {
	fields := DefaultMessageFields()

	if !fields.Envelope {
		t.Error("Expected Envelope to be true in default fields")
	}

	if !fields.BodyPeek {
		t.Error("Expected BodyPeek to be true in default fields")
	}

	expectedHeaders := []string{"From", "Subject", "Date"}
	if !reflect.DeepEqual(fields.Headers, expectedHeaders) {
		t.Errorf("Expected headers %v, got %v", expectedHeaders, fields.Headers)
	}
}

func TestBuildFetchItems(t *testing.T) {
	tests := []struct {
		name          string
		fields        MessageFields
		expectedCount int
		checkForItems []imap.FetchItem
	}{
		{
			name:          "All fields",
			fields:        MessageFields{All: true},
			expectedCount: 1,
			checkForItems: []imap.FetchItem{imap.FetchAll},
		},
		{
			name: "Multiple fields",
			fields: MessageFields{
				Envelope: true,
				Flags:    true,
				Size:     true,
				UID:      true,
			},
			expectedCount: 4,
			checkForItems: []imap.FetchItem{
				imap.FetchEnvelope,
				imap.FetchFlags,
				imap.FetchRFC822Size,
				imap.FetchUid,
			},
		},
		{
			name: "With headers",
			fields: MessageFields{
				Headers: []string{"Subject", "From"},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := buildFetchItems(tt.fields)

			if len(items) != tt.expectedCount {
				t.Errorf("Expected %d items, got %d", tt.expectedCount, len(items))
			}

			if tt.checkForItems != nil {
				for _, expectedItem := range tt.checkForItems {
					found := false
					for _, item := range items {
						if item == expectedItem {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected to find item %v in fetch items", expectedItem)
					}
				}
			}
		})
	}
}

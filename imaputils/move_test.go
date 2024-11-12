package imaputils

import (
	"crypto/tls"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

// MockIMAPClientMove implements IMAPClient interface for testing
type MockIMAPClientMove struct {
	mock.Mock
}

func (m *MockIMAPClientMove) Capability() (map[string]bool, error) {
	args := m.Called()
	return args.Get(0).(map[string]bool), args.Error(1)
}

func (m *MockIMAPClientMove) Create(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockIMAPClientMove) Expunge(ch chan uint32) error {
	args := m.Called(ch)
	// Always close the channel to prevent hanging
	close(ch)
	return args.Error(0)
}

func (m *MockIMAPClientMove) Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	args := m.Called(seqset, items, ch)
	// Always close the channel to prevent hanging
	close(ch)
	return args.Error(1)
}

func (m *MockIMAPClientMove) GetClient() *client.Client {
	args := m.Called()
	if ret := args.Get(0); ret != nil {
		return ret.(*client.Client)
	}
	return nil
}

func (m *MockIMAPClientMove) List(ref string, name string, ch chan *imap.MailboxInfo) error {
	args := m.Called(ref, name, ch)
	if fn, ok := args.Get(0).(func(chan *imap.MailboxInfo)); ok {
		fn(ch)
	}
	close(ch) // Ensure channel is closed
	return args.Error(1)
}

func (m *MockIMAPClientMove) Login(username string, password string) error {
	args := m.Called(username, password)
	return args.Error(0)
}

func (m *MockIMAPClientMove) Logout() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockIMAPClientMove) Select(name string, readOnly bool) (*imap.MailboxStatus, error) {
	args := m.Called(name, readOnly)
	if ret := args.Get(0); ret != nil {
		return ret.(*imap.MailboxStatus), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockIMAPClientMove) UidFetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	args := m.Called(seqset, items, ch)
	if fn, ok := args.Get(0).(func(chan *imap.Message)); ok {
		fn(ch)
	}
	close(ch) // Ensure channel is closed
	return args.Error(1)
}

func (m *MockIMAPClientMove) UidMove(seqSet *imap.SeqSet, mailbox string) error {
	args := m.Called(seqSet, mailbox)
	return args.Error(0)
}

func (m *MockIMAPClientMove) UidSearch(criteria *imap.SearchCriteria) ([]uint32, error) {
	args := m.Called(criteria)
	if ret := args.Get(0); ret != nil {
		return ret.([]uint32), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockIMAPClientMove) UidStore(seqSet *imap.SeqSet, item imap.StoreItem, flags []interface{}, ch chan *imap.Message) error {
	args := m.Called(seqSet, item, flags, ch)
	if fn, ok := args.Get(0).(func(chan *imap.Message)); ok {
		fn(ch)
	}
	close(ch) // Ensure channel is closed
	return args.Error(1)
}

// MockIMAPDialerMove implements IMAPDialer interface for testing
type MockIMAPDialerMove struct {
	mock.Mock
}

func (m *MockIMAPDialerMove) Dial(address string) (IMAPClient, error) {
	args := m.Called(address)
	return args.Get(0).(IMAPClient), args.Error(1)
}

func (m *MockIMAPDialerMove) DialTLS(address string, config *tls.Config) (IMAPClient, error) {
	args := m.Called(address, config)
	return args.Get(0).(IMAPClient), args.Error(1)
}

// Test cases
func TestMoveMessages(t *testing.T) {
	tests := []struct {
		name          string
		messages      []*imap.Message
		sourceFolder  string
		destFolder    string
		batchSize     int
		setupMocks    func(*MockIMAPClientMove, *MockIMAPDialerMove)
		expectedError string
	}{
		{
			name: "successful move",
			messages: []*imap.Message{
				{Uid: 1},
				{Uid: 2},
			},
			sourceFolder: "INBOX",
			destFolder:   "Archive",
			batchSize:    1,
			setupMocks: func(client *MockIMAPClientMove, dialer *MockIMAPDialerMove) {
				// Initial connection setup
				dialer.On("Dial", mock.Anything).Return(client, nil)

				// First login
				client.On("Login", mock.Anything, mock.Anything).Return(nil).Times(2)

				// Folder operations
				client.On("Select", "INBOX", false).Return(&imap.MailboxStatus{}, nil).Times(2)

				// Check destination folder exists
				client.On("List", "", "Archive", mock.Anything).Return(
					func(ch chan *imap.MailboxInfo) {
						ch <- &imap.MailboxInfo{Name: "Archive"}
					},
					nil,
				)

				// Move operations for each message (since batchSize is 1)
				client.On("UidMove", mock.Anything, "Archive").Return(nil).Times(2)

				// Verification operations
				client.On("UidFetch", mock.Anything, []imap.FetchItem{imap.FetchUid}, mock.Anything).Return(
					func(ch chan *imap.Message) {},
					nil,
				).Times(2)

				client.On("Logout").Return(nil).Times(2)
			},
			expectedError: "",
		},
		{
			name: "connection failure",
			messages: []*imap.Message{
				{Uid: 1},
			},
			sourceFolder: "INBOX",
			destFolder:   "Archive",
			batchSize:    1,
			setupMocks: func(client *MockIMAPClientMove, dialer *MockIMAPDialerMove) {
				dialer.On("Dial", mock.Anything).Return(client, fmt.Errorf("connection failed"))
			},
			expectedError: "failed to connect to server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockIMAPClientMove{}
			mockDialer := &MockIMAPDialerMove{}
			tt.setupMocks(mockClient, mockDialer)

			account := Account{
				Server:   "test.example.com",
				User:     "test@example.com",
				Password: "password",
			}

			err := MoveMessages(mockDialer, account, tt.messages, tt.sourceFolder, tt.destFolder, tt.batchSize)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// Add debug output for unmet expectations if assertions fail
			if !mockClient.AssertExpectations(t) {
				t.Log("Actual calls made:")
				for _, call := range mockClient.Calls {
					t.Logf("  %s", call.Method)
				}
			}
			mockDialer.AssertExpectations(t)
		})
	}
}
func TestEnsureFolder(t *testing.T) {
	tests := []struct {
		name          string
		folderName    string
		setupMocks    func(*MockIMAPClientMove, *MockIMAPDialerMove)
		expectedError string
	}{
		{
			name:       "folder already exists",
			folderName: "Archive",
			setupMocks: func(client *MockIMAPClientMove, dialer *MockIMAPDialerMove) {
				dialer.On("Dial", mock.Anything).Return(client, nil)
				client.On("Login", mock.Anything, mock.Anything).Return(nil)
				client.On("List", "", "Archive", mock.Anything).Return(
					func(ch chan *imap.MailboxInfo) {
						ch <- &imap.MailboxInfo{Name: "Archive"}
					},
					nil,
				)
				client.On("Logout").Return(nil)
			},
			expectedError: "",
		},
		{
			name:       "create nested folder",
			folderName: "Parent/Child",
			setupMocks: func(client *MockIMAPClientMove, dialer *MockIMAPDialerMove) {
				dialer.On("Dial", mock.Anything).Return(client, nil)
				client.On("Login", mock.Anything, mock.Anything).Return(nil)

				// Check for full path
				client.On("List", "", "Parent/Child", mock.Anything).Return(
					func(ch chan *imap.MailboxInfo) {},
					nil,
				)

				// Check for parent
				client.On("List", "", "Parent", mock.Anything).Return(
					func(ch chan *imap.MailboxInfo) {},
					nil,
				)

				// Create parent and child
				client.On("Create", "Parent").Return(nil)
				client.On("Create", "Parent/Child").Return(nil)

				client.On("Logout").Return(nil)
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockIMAPClientMove{}
			mockDialer := &MockIMAPDialerMove{}
			tt.setupMocks(mockClient, mockDialer)

			account := Account{
				Server:   "test.example.com",
				User:     "test@example.com",
				Password: "password",
			}

			err := EnsureFolder(mockDialer, account, tt.folderName)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
			mockDialer.AssertExpectations(t)
		})
	}
}

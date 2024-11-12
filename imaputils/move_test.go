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
	// Only close the channel if it's not nil
	if ch != nil {
		close(ch)
	}
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

func (m *MockIMAPClientMove) UidCopy(seqset *imap.SeqSet, mailbox string) error {
	args := m.Called(seqset, mailbox)
	return args.Error(0)
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

// And in UidStore, let's be extra explicit about channel handling:
func (m *MockIMAPClientMove) UidStore(seqSet *imap.SeqSet, item imap.StoreItem, flags []interface{}, ch chan *imap.Message) error {
	args := m.Called(seqSet, item, flags, ch)
	// Only close the channel if it's not nil
	if ch != nil {
		if fn, ok := args.Get(0).(func(chan *imap.Message)); ok {
			fn(ch)
		} else {
			close(ch)
		}
	}
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
		account       Account
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
			account: Account{
				Server:   "test.example.com",
				User:     "test@example.com",
				Password: "password",
			},
			setupMocks: func(client *MockIMAPClientMove, dialer *MockIMAPDialerMove) {
				// We expect multiple connections:
				// 1. Initial connection for checks
				// 2. Connection for EnsureFolder
				// 3. One connection per batch (2 messages, batch size 1 = 2 connections)
				// 4. Final connection for verification
				// Total: 5 connections

				// Setup dialer expectations
				dialer.On("Dial", mock.Anything).Return(client, nil).Times(5)

				// Setup login expectations for all connections
				client.On("Login", mock.Anything, mock.Anything).Return(nil).Times(5)

				// Select mailbox operations
				client.On("Select", "INBOX", false).Return(&imap.MailboxStatus{}, nil).Times(4)

				// Check destination folder exists (called during EnsureFolder)
				client.On("List", "", "Archive", mock.Anything).Return(
					func(ch chan *imap.MailboxInfo) {
						ch <- &imap.MailboxInfo{Name: "Archive"}
					},
					nil,
				)

				// Move operations for each batch (one message per batch)
				client.On("UidMove", mock.MatchedBy(func(seqSet *imap.SeqSet) bool {
					return true // Add more specific validation if needed
				}), "Archive").Return(nil).Times(2)

				// Verification operations for each message
				client.On("UidFetch", mock.MatchedBy(func(seqSet *imap.SeqSet) bool {
					return true // Add more specific validation if needed
				}), []imap.FetchItem{imap.FetchUid}, mock.Anything).Return(
					func(ch chan *imap.Message) {},
					nil,
				).Times(2)

				// Logout for each connection
				client.On("Logout").Return(nil).Times(5)
			},
			expectedError: "",
		},
		{
			name: "move to gmail trash",
			messages: []*imap.Message{
				{Uid: 1},
				{Uid: 2},
			},
			sourceFolder: "INBOX",
			destFolder:   "[Gmail]/Trash",
			batchSize:    1,
			account: Account{
				Server:   "imap.gmail.com",
				User:     "test@gmail.com",
				Password: "password",
			},
			setupMocks: func(client *MockIMAPClientMove, dialer *MockIMAPDialerMove) {
				// For Gmail trash moves, we only need one connection
				dialer.On("Dial", mock.Anything).Return(client, nil)
				client.On("Login", mock.Anything, mock.Anything).Return(nil)
				client.On("Select", "INBOX", false).Return(&imap.MailboxStatus{}, nil)
				client.On("UidCopy", mock.MatchedBy(func(s *imap.SeqSet) bool {
					return true
				}), "[Gmail]/Trash").Return(nil)
				client.On("UidStore",
					mock.MatchedBy(func(s *imap.SeqSet) bool { return true }),
					imap.FormatFlagsOp(imap.AddFlags, true),
					[]interface{}{imap.DeletedFlag},
					(chan *imap.Message)(nil),
				).Return(nil, nil)
				client.On("Expunge", (chan uint32)(nil)).Return(nil)
				client.On("Logout").Return(nil).Once()
			},
			expectedError: "",
		},
		{
			name: "gmail trash copy failure",
			messages: []*imap.Message{
				{Uid: 1},
			},
			sourceFolder: "INBOX",
			destFolder:   "[Gmail]/Trash",
			batchSize:    1,
			account: Account{
				Server:   "imap.gmail.com",
				User:     "test@gmail.com",
				Password: "password",
			},
			setupMocks: func(client *MockIMAPClientMove, dialer *MockIMAPDialerMove) {
				dialer.On("Dial", mock.Anything).Return(client, nil)
				client.On("Login", mock.Anything, mock.Anything).Return(nil)
				client.On("Select", "INBOX", false).Return(&imap.MailboxStatus{}, nil)
				client.On("UidCopy", mock.Anything, "[Gmail]/Trash").Return(fmt.Errorf("copy failed"))
				// Even in failure case, we should expect a logout
				client.On("Logout").Return(nil).Once()
			},
			expectedError: "failed to copy messages to trash",
		},
		{
			name: "gmail trash store flags failure",
			messages: []*imap.Message{
				{Uid: 1},
			},
			sourceFolder: "INBOX",
			destFolder:   "[Gmail]/Trash",
			batchSize:    1,
			account: Account{
				Server:   "imap.gmail.com",
				User:     "test@gmail.com",
				Password: "password",
			},
			setupMocks: func(client *MockIMAPClientMove, dialer *MockIMAPDialerMove) {
				dialer.On("Dial", mock.Anything).Return(client, nil)
				client.On("Login", mock.Anything, mock.Anything).Return(nil)
				client.On("Select", "INBOX", false).Return(&imap.MailboxStatus{}, nil)
				client.On("UidCopy", mock.Anything, "[Gmail]/Trash").Return(nil)
				client.On("UidStore",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(nil, fmt.Errorf("store failed"))
				// Even in failure case, we should expect a logout
				client.On("Logout").Return(nil).Once()
			},
			expectedError: "failed to flag messages as deleted",
		},
		{
			name: "gmail trash expunge failure",
			messages: []*imap.Message{
				{Uid: 1},
			},
			sourceFolder: "INBOX",
			destFolder:   "[Gmail]/Trash",
			batchSize:    1,
			account: Account{
				Server:   "imap.gmail.com",
				User:     "test@gmail.com",
				Password: "password",
			},
			setupMocks: func(client *MockIMAPClientMove, dialer *MockIMAPDialerMove) {
				dialer.On("Dial", mock.Anything).Return(client, nil)
				client.On("Login", mock.Anything, mock.Anything).Return(nil)
				client.On("Select", "INBOX", false).Return(&imap.MailboxStatus{}, nil)
				client.On("UidCopy", mock.Anything, "[Gmail]/Trash").Return(nil)
				client.On("UidStore",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(nil, nil)
				client.On("Expunge", mock.Anything).Return(fmt.Errorf("expunge failed"))
				// Even in failure case, we should expect a logout
				client.On("Logout").Return(nil).Once()
			},
			expectedError: "failed to expunge messages",
		},
		{
			name: "connection failure",
			messages: []*imap.Message{
				{Uid: 1},
			},
			sourceFolder: "INBOX",
			destFolder:   "Archive",
			batchSize:    1,
			account: Account{
				Server:   "test.example.com",
				User:     "test@example.com",
				Password: "password",
			},
			setupMocks: func(client *MockIMAPClientMove, dialer *MockIMAPDialerMove) {
				// No Logout expectation needed here since connection fails
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

			err := MoveMessages(mockDialer, tt.account, tt.messages, tt.sourceFolder, tt.destFolder, tt.batchSize)

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
					t.Logf("  %s(%v)", call.Method, call.Arguments)
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

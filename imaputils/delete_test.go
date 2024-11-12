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

// Mocking IMAPDialer
type MockIMAPDialer struct {
	mock.Mock
}

func (m *MockIMAPDialer) Dial(account string) (IMAPClient, error) {
	args := m.Called(account)
	return args.Get(0).(IMAPClient), args.Error(1)
}

func (m *MockIMAPDialer) DialTLS(address string, config *tls.Config) (IMAPClient, error) {
	args := m.Called(address, config)
	return args.Get(0).(IMAPClient), args.Error(1)
}

// Mocking IMAPClient
type MockIMAPClient struct {
	mock.Mock
}

func (m *MockIMAPClient) Capability() (map[string]bool, error) {
	args := m.Called()
	return args.Get(0).(map[string]bool), args.Error(1)
}

func (m *MockIMAPClient) Create(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockIMAPClient) Expunge(ch chan uint32) error {
	args := m.Called(ch)
	return args.Error(0)
}

func (m *MockIMAPClient) Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	args := m.Called(seqset, items, ch)
	return args.Error(0)
}

func (m *MockIMAPClient) GetClient() *client.Client {
	args := m.Called()
	return args.Get(0).(*client.Client)
}

func (m *MockIMAPClient) List(ref string, name string, ch chan *imap.MailboxInfo) error {
	args := m.Called(ref, name, ch)
	return args.Error(0)
}

func (m *MockIMAPClient) Login(username, password string) error {
	args := m.Called(username, password)
	return args.Error(0)
}

func (m *MockIMAPClient) Logout() error {
	return m.Called().Error(0)
}

func (m *MockIMAPClient) Select(name string, readOnly bool) (*imap.MailboxStatus, error) {
	args := m.Called(name, readOnly)
	return args.Get(0).(*imap.MailboxStatus), args.Error(1)
}

func (m *MockIMAPClient) UidCopy(seqset *imap.SeqSet, mailbox string) error {
	args := m.Called(seqset, mailbox)
	return args.Error(0)
}

func (m *MockIMAPClient) UidFetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	args := m.Called(seqset, items, ch)
	return args.Error(0)
}

func (m *MockIMAPClient) UidMove(seqset *imap.SeqSet, mailbox string) error {
	args := m.Called(seqset, mailbox)
	return args.Error(0)
}

func (m *MockIMAPClient) UidSearch(criteria *imap.SearchCriteria) ([]uint32, error) {
	args := m.Called(criteria)
	return args.Get(0).([]uint32), args.Error(1)
}

func (m *MockIMAPClient) UidStore(seqset *imap.SeqSet, item imap.StoreItem, flags []interface{}, ch chan *imap.Message) error {
	args := m.Called(seqset, item, flags, ch)
	return args.Error(0)
}

func TestDeleteMessages_NoMessages(t *testing.T) {
	dialer := new(MockIMAPDialer)
	account := Account{Purge: false}
	messages := []*imap.Message{}

	client := new(MockIMAPClient)
	dialer.On("Dial", mock.Anything).Return(client, nil)
	client.On("List", mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ch := args.Get(2).(chan *imap.MailboxInfo)
		close(ch) // Close the channel to simulate end of data
	})

	err := DeleteMessages(dialer, account, messages, "INBOX")
	assert.NoError(t, err)
}

func TestDeleteMessages_MoveToTrash(t *testing.T) {
	dialer := new(MockIMAPDialer)
	client := new(MockIMAPClient)
	account := Account{Purge: false}
	messages := []*imap.Message{{
		SeqNum: 1,
		Uid:    123,
	}}

	// Basic setup
	dialer.On("Dial", mock.Anything).Return(client, nil)
	client.On("Login", mock.Anything, mock.Anything).Return(nil)
	client.On("Logout").Return(nil)

	// ListFolders operation to find trash folder
	client.On("List", "", "*", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ch := args.Get(2).(chan *imap.MailboxInfo)
		ch <- &imap.MailboxInfo{Name: "INBOX"}
		ch <- &imap.MailboxInfo{Name: "Deleted Items"}
		close(ch)
	})

	// EnsureFolder verification
	client.On("List", "", "Deleted Items", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ch := args.Get(2).(chan *imap.MailboxInfo)
		ch <- &imap.MailboxInfo{Name: "Deleted Items"}
		close(ch)
	})

	// Select operations
	client.On("Select", mock.Anything, mock.Anything).Return(&imap.MailboxStatus{}, nil)

	// Move and verify
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(messages[0].Uid)
	client.On("UidMove", seqSet, "Deleted Items").Return(nil)
	client.On("UidFetch", mock.Anything, []imap.FetchItem{imap.FetchUid}, mock.Anything).Return(nil).
		Run(func(args mock.Arguments) {
			ch := args.Get(2).(chan *imap.Message)
			close(ch)
		})

	err := DeleteMessages(dialer, account, messages, "INBOX")
	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func TestDeleteMessages_NoExistingTrashFolder(t *testing.T) {
	dialer := new(MockIMAPDialer)
	client := new(MockIMAPClient)
	account := Account{Purge: false}
	messages := []*imap.Message{{
		SeqNum: 1,
		Uid:    123,
	}}

	// Basic setup
	dialer.On("Dial", mock.Anything).Return(client, nil)
	client.On("Login", mock.Anything, mock.Anything).Return(nil)
	client.On("Logout").Return(nil)

	// ListFolders operation to find trash folder
	client.On("List", "", "*", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ch := args.Get(2).(chan *imap.MailboxInfo)
		ch <- &imap.MailboxInfo{Name: "INBOX"}
		// Don't include Deleted Items
		close(ch)
	})

	// EnsureFolder verification
	client.On("List", "", "Deleted Items", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ch := args.Get(2).(chan *imap.MailboxInfo)
		// Don't include Deleted Items - it doesn't exist yet
		close(ch)
	})

	// Folder creation
	client.On("Create", "Deleted Items").Return(nil)

	// Select operations - allow any select calls
	client.On("Select", mock.Anything, mock.Anything).Return(&imap.MailboxStatus{}, nil)

	// Move and verify
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(messages[0].Uid)
	client.On("UidMove", seqSet, "Deleted Items").Return(nil)
	client.On("UidFetch", mock.Anything, []imap.FetchItem{imap.FetchUid}, mock.Anything).Return(nil).
		Run(func(args mock.Arguments) {
			ch := args.Get(2).(chan *imap.Message)
			close(ch)
		})

	err := DeleteMessages(dialer, account, messages, "INBOX")
	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func TestDeleteMessages_CreateTrashFolder(t *testing.T) {
	dialer := new(MockIMAPDialer)
	client := new(MockIMAPClient)
	account := Account{Purge: false}
	messages := []*imap.Message{{
		SeqNum: 1,
		Uid:    123,
	}}

	dialer.On("Dial", mock.Anything).Return(client, nil)
	client.On("Login", mock.Anything, mock.Anything).Return(nil)

	// Mock empty folder list
	client.On("List", mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ch := args.Get(2).(chan *imap.MailboxInfo)
		close(ch)
	})

	// Mock mailbox operations with more flexible matching
	client.On("Select", mock.Anything, mock.Anything).Return(&imap.MailboxStatus{}, nil)
	client.On("Create", mock.Anything).Return(nil)

	// Mock UidFetch for message verification
	client.On("UidFetch", mock.Anything, []imap.FetchItem{imap.FetchUid}, mock.Anything).Return(nil).
		Run(func(args mock.Arguments) {
			ch := args.Get(2).(chan *imap.Message)
			close(ch)
		})

	// Mock the move operation
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(messages[0].Uid)
	client.On("UidMove", mock.Anything, mock.Anything).Return(nil)

	client.On("Logout").Return(nil)

	err := DeleteMessages(dialer, account, messages, "INBOX")
	assert.NoError(t, err)

	dialer.AssertExpectations(t)
	client.AssertExpectations(t)
}

func TestDeleteMessages_PurgeMessages(t *testing.T) {
	dialer := new(MockIMAPDialer)
	client := new(MockIMAPClient)
	account := Account{Purge: true}
	messages := []*imap.Message{{SeqNum: 1, Uid: 123}}

	dialer.On("Dial", mock.Anything).Return(client, nil)
	client.On("Login", mock.Anything, mock.Anything).Return(nil)

	// Mock selecting the source folder
	client.On("Select", mock.Anything, mock.Anything).Return(&imap.MailboxStatus{}, nil)

	// Mock the message deletion
	client.On("UidStore", mock.Anything, mock.Anything, []interface{}{imap.DeletedFlag}, mock.Anything).Return(nil)
	client.On("Expunge", mock.Anything).Return(nil)
	client.On("Logout").Return(nil)

	err := DeleteMessages(dialer, account, messages, "INBOX")
	assert.NoError(t, err)

	dialer.AssertExpectations(t)
	client.AssertExpectations(t)
}

func TestDeleteMessages_ErrorHandling(t *testing.T) {
	// Test dial error
	dialer := new(MockIMAPDialer)
	client := new(MockIMAPClient)
	account := Account{Purge: false}
	messages := []*imap.Message{{SeqNum: 1, Uid: 123}}

	dialError := fmt.Errorf("connection failed")
	dialer.On("Dial", mock.Anything).Return(client, dialError)

	err := DeleteMessages(dialer, account, messages, "INBOX")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection failed")

	// Test folder creation error
	dialer = new(MockIMAPDialer)
	client = new(MockIMAPClient)

	dialer.On("Dial", mock.Anything).Return(client, nil)
	client.On("Login", mock.Anything, mock.Anything).Return(nil)
	client.On("List", mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ch := args.Get(2).(chan *imap.MailboxInfo)
		close(ch)
	})
	client.On("Select", mock.Anything, mock.Anything).Return(&imap.MailboxStatus{}, nil)
	createError := fmt.Errorf("failed to create folder")
	client.On("Create", mock.Anything).Return(createError)
	client.On("Logout").Return(nil)

	err = DeleteMessages(dialer, account, messages, "INBOX")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create folder")

	dialer.AssertExpectations(t)
	client.AssertExpectations(t)
}

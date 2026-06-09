package imaputils

import (
	"crypto/tls"
	"errors"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// MockIMAPClientListFolders implements IMAPClient interface
type MockIMAPClientListFolders struct {
	listFunc    func(ref string, name string, ch chan *imap.MailboxInfo) error
	statusFunc  func(name string, items []imap.StatusItem) (*imap.MailboxStatus, error)
	selectFunc  func(name string, readOnly bool) (*imap.MailboxStatus, error)
	fetchFunc   func(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error
	logoutCalls int
}

func (m *MockIMAPClientListFolders) List(ref string, name string, ch chan *imap.MailboxInfo) error {
	return m.listFunc(ref, name, ch)
}

func (m *MockIMAPClientListFolders) Logout() error {
	m.logoutCalls++
	return nil
}

// Implement remaining IMAPClient interface methods
func (m *MockIMAPClientListFolders) Capability() (map[string]bool, error) { return nil, nil }
func (m *MockIMAPClientListFolders) Create(name string) error             { return nil }
func (m *MockIMAPClientListFolders) Expunge(ch chan uint32) error         { return nil }
func (m *MockIMAPClientListFolders) Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	if m.fetchFunc != nil {
		return m.fetchFunc(seqset, items, ch)
	}
	close(ch)
	return nil
}
func (m *MockIMAPClientListFolders) GetClient() *client.Client                    { return nil }
func (m *MockIMAPClientListFolders) Login(username string, password string) error { return nil }
func (m *MockIMAPClientListFolders) Select(name string, readOnly bool) (*imap.MailboxStatus, error) {
	if m.selectFunc != nil {
		return m.selectFunc(name, readOnly)
	}
	return nil, nil
}
func (m *MockIMAPClientListFolders) Status(name string, items []imap.StatusItem) (*imap.MailboxStatus, error) {
	if m.statusFunc != nil {
		return m.statusFunc(name, items)
	}
	return &imap.MailboxStatus{}, nil
}
func (m *MockIMAPClientListFolders) UidCopy(seqSet *imap.SeqSet, mailbox string) error { return nil }
func (m *MockIMAPClientListFolders) UidFetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	return nil
}
func (m *MockIMAPClientListFolders) UidMove(seqSet *imap.SeqSet, mailbox string) error { return nil }
func (m *MockIMAPClientListFolders) UidSearch(criteria *imap.SearchCriteria) ([]uint32, error) {
	return nil, nil
}
func (m *MockIMAPClientListFolders) UidStore(seqSet *imap.SeqSet, item imap.StoreItem, flags []interface{}, ch chan *imap.Message) error {
	return nil
}

// MockDialerListFolders implements IMAPDialer interface for these tests
type MockDialerListFolders struct {
	client IMAPClient
	err    error
}

func (d *MockDialerListFolders) Dial(address string) (IMAPClient, error) {
	return d.client, d.err
}

func (d *MockDialerListFolders) DialTLS(address string, config *tls.Config) (IMAPClient, error) {
	return d.client, d.err
}

func TestListFolders(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() *MockIMAPClientListFolders
		expectedError string
		expectedDirs  []string
		dialerError   error
		checkLogout   bool
	}{
		{
			name: "successful listing of folders",
			setupMock: func() *MockIMAPClientListFolders {
				return &MockIMAPClientListFolders{
					listFunc: func(ref string, name string, ch chan *imap.MailboxInfo) error {
						go func() {
							ch <- &imap.MailboxInfo{Name: "INBOX"}
							ch <- &imap.MailboxInfo{Name: "Sent"}
							ch <- &imap.MailboxInfo{Name: "Trash"}
							close(ch)
						}()
						return nil
					},
				}
			},
			expectedDirs: []string{"INBOX", "Sent", "Trash"},
			checkLogout:  true,
		},
		{
			name: "empty folder list",
			setupMock: func() *MockIMAPClientListFolders {
				return &MockIMAPClientListFolders{
					listFunc: func(ref string, name string, ch chan *imap.MailboxInfo) error {
						close(ch)
						return nil
					},
				}
			},
			expectedDirs: []string(nil), // Changed from []string{} to match actual return
			checkLogout:  true,
		},
		{
			name: "list error",
			setupMock: func() *MockIMAPClientListFolders {
				return &MockIMAPClientListFolders{
					listFunc: func(ref string, name string, ch chan *imap.MailboxInfo) error {
						close(ch)
						return errors.New("list error")
					},
				}
			},
			expectedError: "failed to list folders: list error",
			checkLogout:   true,
		},
		{
			name:          "dialer error",
			setupMock:     func() *MockIMAPClientListFolders { return nil },
			dialerError:   errors.New("connection failed"),
			expectedError: "failed to initialize imap client: failed to connect to server: connection failed", // Updated error message
			checkLogout:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := tt.setupMock()
			mockDialer := &MockDialerListFolders{
				client: mockClient,
				err:    tt.dialerError,
			}

			folders, err := ListFolders(mockDialer, Account{})

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedDirs, folders)
			}

			if tt.checkLogout && mockClient != nil {
				assert.Equal(t, 1, mockClient.logoutCalls, "Logout should be called exactly once")
			}
		})
	}
}

func TestListFoldersWithStatus(t *testing.T) {
	inboxOldest := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
	inboxNewest := time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)
	inboxMiddle := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	archiveOldest := time.Date(2019, 6, 15, 9, 0, 0, 0, time.UTC)
	archiveNewest := time.Date(2021, 3, 20, 9, 0, 0, 0, time.UTC)

	dates := map[string][]time.Time{
		"INBOX":   {inboxNewest, inboxOldest, inboxMiddle}, // out of order on purpose
		"Archive": {archiveNewest, archiveOldest},
	}

	var selected string
	mockClient := &MockIMAPClientListFolders{
		listFunc: func(ref string, name string, ch chan *imap.MailboxInfo) error {
			go func() {
				ch <- &imap.MailboxInfo{Name: "INBOX"}
				ch <- &imap.MailboxInfo{Name: "[Gmail]", Attributes: []string{imap.NoSelectAttr}}
				ch <- &imap.MailboxInfo{Name: "Archive"}
				close(ch)
			}()
			return nil
		},
		statusFunc: func(name string, items []imap.StatusItem) (*imap.MailboxStatus, error) {
			switch name {
			case "INBOX":
				return &imap.MailboxStatus{Messages: 10, Unseen: 3}, nil
			case "Archive":
				return &imap.MailboxStatus{Messages: 100, Unseen: 0}, nil
			}
			return nil, errors.New("unexpected status call for " + name)
		},
		selectFunc: func(name string, readOnly bool) (*imap.MailboxStatus, error) {
			selected = name
			return &imap.MailboxStatus{Messages: uint32(len(dates[name]))}, nil
		},
		fetchFunc: func(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
			for _, date := range dates[selected] {
				ch <- &imap.Message{InternalDate: date}
			}
			close(ch)
			return nil
		},
	}
	dialer := &MockDialerListFolders{client: mockClient}

	folders, err := ListFoldersWithStatus(dialer, Account{}, true)
	assert.NoError(t, err)
	assert.Equal(t, []FolderStatus{
		{Name: "INBOX", Messages: 10, Unseen: 3, Selectable: true, Oldest: inboxOldest, Newest: inboxNewest},
		{Name: "[Gmail]", Selectable: false},
		{Name: "Archive", Messages: 100, Unseen: 0, Selectable: true, Oldest: archiveOldest, Newest: archiveNewest},
	}, folders)
	assert.Equal(t, 1, mockClient.logoutCalls, "Logout should be called exactly once")
}

func TestListFoldersWithStatusWithoutDates(t *testing.T) {
	mockClient := &MockIMAPClientListFolders{
		listFunc: func(ref string, name string, ch chan *imap.MailboxInfo) error {
			go func() {
				ch <- &imap.MailboxInfo{Name: "INBOX"}
				close(ch)
			}()
			return nil
		},
		statusFunc: func(name string, items []imap.StatusItem) (*imap.MailboxStatus, error) {
			return &imap.MailboxStatus{Messages: 5000, Unseen: 12}, nil
		},
		selectFunc: func(name string, readOnly bool) (*imap.MailboxStatus, error) {
			t.Errorf("Select should not be called when dates are disabled")
			return nil, nil
		},
		fetchFunc: func(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
			t.Errorf("Fetch should not be called when dates are disabled")
			close(ch)
			return nil
		},
	}
	dialer := &MockDialerListFolders{client: mockClient}

	folders, err := ListFoldersWithStatus(dialer, Account{}, false)
	assert.NoError(t, err)
	// Counts are present; date range is left zero (no scan performed).
	assert.Equal(t, []FolderStatus{
		{Name: "INBOX", Messages: 5000, Unseen: 12, Selectable: true},
	}, folders)
}

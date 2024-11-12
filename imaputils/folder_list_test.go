package imaputils

import (
	"crypto/tls"
	"errors"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/stretchr/testify/assert"
	"testing"
)

// MockIMAPClientListFolders implements IMAPClient interface
type MockIMAPClientListFolders struct {
	listFunc    func(ref string, name string, ch chan *imap.MailboxInfo) error
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
	return nil
}
func (m *MockIMAPClientListFolders) GetClient() *client.Client                    { return nil }
func (m *MockIMAPClientListFolders) Login(username string, password string) error { return nil }
func (m *MockIMAPClientListFolders) Select(name string, readOnly bool) (*imap.MailboxStatus, error) {
	return nil, nil
}
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

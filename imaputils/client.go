package imaputils

import (
	"crypto/tls"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/wryfi/shemail/logging"
)

var log = &logging.Logger

// Account represents the fields that define an IMAP account
type Account struct {
	Name     string // identifier for the account
	User     string
	Password string
	Server   string
	Port     int
	TLS      bool
	Purge    bool
	Default  bool
}

// IMAPClient defines the minimal interface for IMAP client operations
type IMAPClient interface {
	Capability() (map[string]bool, error)
	Create(name string) error
	Expunge(ch chan uint32) error
	Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error
	GetClient() *client.Client
	List(ref string, name string, ch chan *imap.MailboxInfo) error
	Login(username string, password string) error
	Logout() error
	Select(name string, readOnly bool) (*imap.MailboxStatus, error)
	UidCopy(seqset *imap.SeqSet, dest string) error
	UidFetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error
	UidMove(seqSet *imap.SeqSet, mailbox string) error
	UidSearch(criteria *imap.SearchCriteria) (uids []uint32, err error)
	UidStore(seqSet *imap.SeqSet, item imap.StoreItem, flags []interface{}, ch chan *imap.Message) error
}

// ShemailClient represents the concrete implementation of the IMAPClient
type ShemailClient struct {
	Client *client.Client
}

// Ensure ShemailClient implements IMAPClient interface
var _ IMAPClient = &ShemailClient{}

func (c *ShemailClient) Capability() (map[string]bool, error) {
	return c.Client.Capability()
}

func (c *ShemailClient) Create(name string) error {
	return c.Client.Create(name)
}

func (c *ShemailClient) Expunge(ch chan uint32) error {
	return c.Client.Expunge(ch)
}

func (c *ShemailClient) Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	return c.Client.Fetch(seqset, items, ch)
}

func (c *ShemailClient) GetClient() *client.Client {
	return c.Client
}

func (c *ShemailClient) List(ref string, name string, ch chan *imap.MailboxInfo) error {
	return c.Client.List(ref, name, ch)
}

func (c *ShemailClient) Login(username string, password string) error {
	return c.Client.Login(username, password)
}

func (c *ShemailClient) Logout() error {
	return c.Client.Logout()
}

func (c *ShemailClient) Select(name string, readOnly bool) (*imap.MailboxStatus, error) {
	return c.Client.Select(name, readOnly)
}

func (c *ShemailClient) UidCopy(seqset *imap.SeqSet, dest string) error {
	return c.Client.UidCopy(seqset, dest)
}

func (c *ShemailClient) UidFetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	return c.Client.UidFetch(seqset, items, ch)
}

func (c *ShemailClient) UidMove(seqSet *imap.SeqSet, mailbox string) error {
	return c.Client.UidMove(seqSet, mailbox)
}

func (c *ShemailClient) UidSearch(criteria *imap.SearchCriteria) (uids []uint32, err error) {
	return c.Client.UidSearch(criteria)
}

func (c *ShemailClient) UidStore(seqSet *imap.SeqSet, item imap.StoreItem, flags []interface{}, ch chan *imap.Message) error {
	return c.Client.UidStore(seqSet, item, flags, ch)
}

// IMAPDialer defines the interface for establishing an IMAP connection
type IMAPDialer interface {
	Dial(address string) (IMAPClient, error)
	DialTLS(address string, config *tls.Config) (IMAPClient, error)
}

// SheMailDialer will handle the connection methods
type SheMailDialer struct{}

// Ensure SheMailDialer implements IMAPDialer interface
var _ IMAPDialer = &SheMailDialer{}

func (d *SheMailDialer) Dial(address string) (IMAPClient, error) {
	c, err := client.Dial(address)
	if err != nil {
		return nil, err
	}
	return &ShemailClient{Client: c}, nil
}

func (d *SheMailDialer) DialTLS(address string, config *tls.Config) (IMAPClient, error) {
	c, err := client.DialTLS(address, config)
	if err != nil {
		return nil, err
	}
	return &ShemailClient{Client: c}, nil
}

// getImapClient returns an authenticated IMAP client for the given account
func getImapClient(dialer IMAPDialer, account Account) (IMAPClient, error) {
	var imapClient IMAPClient
	serverPort := fmt.Sprintf("%s:%d", account.Server, account.Port)

	var connectionError error
	if account.TLS {
		imapClient, connectionError = dialer.DialTLS(serverPort, &tls.Config{})
	} else {
		imapClient, connectionError = dialer.Dial(serverPort)
	}
	if connectionError != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", connectionError)
	}

	if err := imapClient.Login(account.User, account.Password); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	return imapClient, nil
}

// connectToMailbox returns an authenticated IMAP client for the given account and folder
func connectToMailbox(dialer IMAPDialer, account Account, folder string, readOnly bool) (IMAPClient, error) {
	// Use getImapClient to establish the connection and authenticate
	imapClient, err := getImapClient(dialer, account)
	if err != nil {
		return nil, fmt.Errorf("failed to get IMAP client: %w", err)
	}

	// Select the specified folder
	_, err = imapClient.Select(folder, readOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to select folder: %w", err)
	}

	return imapClient, nil
}

var SheClient = &ShemailClient{}
var SheDialer = &SheMailDialer{}

package imap

import (
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/spf13/viper"
	"github.com/wryfi/shemail/logging"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var log = &logging.Logger

// SearchOptions represents the optional search parameters
type SearchOptions struct {
	To        *string    // Optional To address
	From      *string    // Optional From address
	Subject   *string    // Optional Subject
	StartDate *time.Time // Optional start date
	EndDate   *time.Time // Optional end date
	Seen      *bool      // Optional seen flag
}

// SenderCount holds information about a sender and the number of messages sent.
type SenderCount struct {
	Sender       string
	MessageCount int
}

func GetImapClient() (*client.Client, error) {
	var connectionError error
	var imapClient *client.Client
	server := viper.GetString("imap.server")
	port := viper.GetInt("imap.port")
	serverPort := fmt.Sprintf("%s:%d", server, port)
	tls := viper.GetBool("imap.tls")

	if tls {
		imapClient, connectionError = client.DialTLS(serverPort, nil)
	} else {
		imapClient, connectionError = client.Dial(serverPort)
	}
	if connectionError != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", connectionError)
	}
	//defer imapClient.Logout()

	user := viper.GetString("imap.user")
	password := viper.GetString("imap.password")

	// Login
	if err := imapClient.Login(user, password); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	return imapClient, nil
}

// ListFolders lists all folders in the IMAP account
func ListFolders(server, username, password string) ([]string, error) {
	// Connect to the server
	imapClient, err := GetImapClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer imapClient.Logout()

	// List mailboxes (folders)
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- imapClient.List("", "*", mailboxes)
	}()

	var folders []string
	for m := range mailboxes {
		folders = append(folders, m.Name)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	return folders, nil
}

func SearchMessages(mailbox string, criteria *imap.SearchCriteria) ([]*imap.Message, error) {
	// Connect to the server
	imapClient, err := GetImapClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer imapClient.Logout()

	if _, err := imapClient.Select(mailbox, false); err != nil {
		return nil, fmt.Errorf("failed to select mailbox: %v", err)
	}

	// Get server capabilities after login
	caps, err := imapClient.Capability()
	if err != nil {
		return nil, fmt.Errorf("failed to get capabilities: %v", err)
	}

	log.Debug().Msgf("Server capabilities:")
	for cap := range caps {
		log.Debug().Msgf("- %s", cap)
	}

	// Check if FUZZY search is supported
	var seqNums []uint32

	log.Debug().Msg("using standard search")
	seqNums, err = imapClient.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("search failed: %v", err)
	}

	if len(seqNums) == 0 {
		return []*imap.Message{}, nil
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(seqNums...)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	var results []*imap.Message
	go func() {
		done <- imapClient.Fetch(seqSet, []imap.FetchItem{
			imap.FetchAll,
		}, messages)
	}()

	for msg := range messages {
		results = append(results, msg)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("fetch failed: %v", err)
	}

	return results, nil
}

// FormatAddress formats an IMAP address into a human-readable string.
func FormatAddress(address *imap.Address) string {
	//var name, mailbox, host string
	var mailbox, host string
	//if address.PersonalName != "" {
	//	name = address.PersonalName
	//}

	if address.MailboxName != "" && address.HostName != "" {
		mailbox = address.MailboxName
		host = address.HostName
	}

	//if name != "" {
	//	return fmt.Sprintf("%s <%s@%s>", name, mailbox, host)
	//}
	return fmt.Sprintf("%s@%s", mailbox, host)
}

// FormatAddresses formats a slice of IMAP addresses into a comma-separated string.
func FormatAddresses(addresses []*imap.Address) []string {
	formatted := []string{}
	for _, addr := range addresses {
		formatted = append(formatted, FormatAddress(addr))
	}
	return formatted
}

// FormatAddressesCSV formats a slice of IMAP addresses into a comma-separated string.
func FormatAddressesCSV(addresses []*imap.Address) string {
	formatted := FormatAddresses(addresses)
	count := len(formatted)
	if count > 1 {
		joined := strings.Join(formatted[0:1], ", ")
		return fmt.Sprintf("%s (+%d)", joined, count-1)
	}
	return strings.Join(formatted, ", ")
}

// BuildSearchCriteria builds search criteria based on the given search options.
func BuildSearchCriteria(opts SearchOptions) *imap.SearchCriteria {
	criteria := &imap.SearchCriteria{
		Header: make(map[string][]string),
	}

	if opts.To != nil {
		criteria.Header["To"] = []string{*opts.To}
		log.Debug().Msgf("Adding To criterion: %s\n", *opts.To)
	}
	if opts.From != nil {
		criteria.Header["From"] = []string{*opts.From}
		log.Debug().Msgf("Adding From criterion: %s\n", *opts.From)
	}
	if opts.Subject != nil {
		criteria.Header["Subject"] = []string{*opts.Subject}
		log.Debug().Msgf("Adding Subject criterion: %s\n", *opts.Subject)
	}

	if opts.StartDate != nil {
		// Set both internal and sent date criteria
		criteria.Since = *opts.StartDate
		criteria.SentSince = *opts.StartDate
		log.Debug().Msgf("Adding start date criteria: %s\n", opts.StartDate.String())
	} else {
		// If no start date is specified, use a very old date
		veryOldDate := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		criteria.Since = veryOldDate
		criteria.SentSince = veryOldDate
		log.Debug().Msgf("No start date specified, using epoch time")
	}

	if opts.EndDate != nil {
		// Set both internal and sent date criteria
		// Add one day to include the end date
		endDatePlusOne := opts.EndDate.AddDate(0, 0, 1)
		criteria.Before = endDatePlusOne
		criteria.SentBefore = endDatePlusOne
		log.Debug().Msgf("Adding end date criteria: %s\n", opts.EndDate.String())
	} else {
		// If no end date is specified, use a future date
		farFuture := time.Date(2999, 12, 31, 23, 59, 59, 0, time.UTC)
		criteria.Before = farFuture
		criteria.SentBefore = farFuture
		log.Debug().Msgf("No end date specified, using far future date")
	}

	if opts.Seen != nil {
		if *opts.Seen {
			criteria.WithFlags = []string{imap.SeenFlag}
			log.Debug().Msgf("Adding Seen criterion")
		} else {
			criteria.WithoutFlags = []string{imap.SeenFlag}
			log.Debug().Msgf("Adding Unseen criterion")
		}
	} else {
		log.Debug().Msgf("No seen/unseen flag set; searching all messages")
	}

	log.Debug().Msgf("Final search criteria: %+v", *criteria)

	return criteria
}

// BuildORSearchCriteria creates an IMAP search criteria based on provided options using OR logic
func BuildORSearchCriteria(opts SearchOptions) *imap.SearchCriteria {
	var criteriaList []*imap.SearchCriteria

	// Add To criteria if specified
	if opts.To != nil {
		c := &imap.SearchCriteria{}
		c.Header.Add("To", *opts.To)
		criteriaList = append(criteriaList, c)
	}

	// Add From criteria if specified
	if opts.From != nil {
		c := &imap.SearchCriteria{}
		c.Header.Add("From", *opts.From)
		criteriaList = append(criteriaList, c)
	}

	// Add Subject criteria if specified
	if opts.Subject != nil {
		c := &imap.SearchCriteria{}
		c.Header.Add("Subject", *opts.Subject)
		criteriaList = append(criteriaList, c)
	}

	// Add date range if specified
	if opts.StartDate != nil && opts.EndDate != nil {
		c := &imap.SearchCriteria{
			Since:  *opts.StartDate,
			Before: *opts.EndDate,
		}
		criteriaList = append(criteriaList, c)
	} else if opts.StartDate != nil {
		c := &imap.SearchCriteria{
			Since: *opts.StartDate,
		}
		criteriaList = append(criteriaList, c)
	} else if opts.EndDate != nil {
		c := &imap.SearchCriteria{
			Before: *opts.EndDate,
		}
		criteriaList = append(criteriaList, c)
	}

	// Add seen/unseen flag if specified
	if opts.Seen != nil {
		c := &imap.SearchCriteria{}
		if *opts.Seen {
			c.WithFlags = []string{imap.SeenFlag}
		} else {
			c.WithoutFlags = []string{imap.SeenFlag}
		}
		criteriaList = append(criteriaList, c)
	}

	// If no criteria were specified, return empty criteria (matches everything)
	if len(criteriaList) == 0 {
		return &imap.SearchCriteria{}
	}

	// If only one criterion was specified, return it directly
	if len(criteriaList) == 1 {
		return criteriaList[0]
	}

	// Combine all criteria with OR
	result := &imap.SearchCriteria{
		Or: [][2]*imap.SearchCriteria{},
	}

	// Build the OR chain
	for i := 0; i < len(criteriaList)-1; i++ {
		result.Or = append(result.Or, [2]*imap.SearchCriteria{
			criteriaList[i],
			criteriaList[i+1],
		})
	}

	return result
}

// MessageFields represents the fields to fetch from IMAP messages
type MessageFields struct {
	Envelope  bool
	Body      bool
	Flags     bool
	Headers   []string // Specific headers to fetch
	BodyPeek  bool     // Whether to mark messages as read when fetching
	Structure bool     // Message structure/MIME parts
	Size      bool     // Message size
	UID       bool     // Message UID
	All       bool     // Fetch all fields (overrides other options)
}

// DefaultMessageFields returns MessageFields with commonly used defaults
func DefaultMessageFields() MessageFields {
	return MessageFields{
		Envelope: true,
		Headers:  []string{"From", "Subject", "Date"},
		BodyPeek: true,
	}
}

// FetchMessages fetches a list of messages from the specified mailbox with customizable field selection.
func FetchMessages(mailbox string, fields MessageFields) ([]*imap.Message, error) {
	imapClient, err := GetImapClient()
	if err != nil {
		return nil, fmt.Errorf("error getting imap client: %w", err)
	}
	defer imapClient.Logout()

	// Select mailbox
	mbox, err := imapClient.Select(mailbox, false)
	if err != nil {
		return nil, fmt.Errorf("failed to select mailbox: %w", err)
	}

	// If mailbox is empty, return early
	if mbox.Messages == 0 {
		return []*imap.Message{}, nil
	}

	// Build fetch items based on requested fields
	items := buildFetchItems(fields)

	// Pre-allocate slice with known capacity
	fetchedMessages := make([]*imap.Message, 0, mbox.Messages)

	// Use batch processing for large mailboxes
	const batchSize = 1000
	for i := uint32(1); i <= mbox.Messages; i += batchSize {
		seqset := new(imap.SeqSet)
		end := i + batchSize - 1
		if end > mbox.Messages {
			end = mbox.Messages
		}
		seqset.AddRange(i, end)

		messages := make(chan *imap.Message, batchSize)
		done := make(chan error, 1)

		go func() {
			done <- imapClient.Fetch(seqset, items, messages)
		}()

		for msg := range messages {
			fetchedMessages = append(fetchedMessages, msg)
		}

		if err := <-done; err != nil {
			return nil, fmt.Errorf("failed to fetch messages batch %d-%d: %w", i, end, err)
		}
	}

	return fetchedMessages, nil
}

// buildFetchItems converts MessageFields into IMAP fetch items
func buildFetchItems(fields MessageFields) []imap.FetchItem {
	if fields.All {
		return []imap.FetchItem{imap.FetchAll}
	}

	items := make([]imap.FetchItem, 0)

	if fields.Envelope {
		items = append(items, imap.FetchEnvelope)
	}

	if fields.Flags {
		items = append(items, imap.FetchFlags)
	}

	if fields.Size {
		items = append(items, imap.FetchRFC822Size)
	}

	if fields.UID {
		items = append(items, imap.FetchUid)
	}

	if fields.Structure {
		items = append(items, imap.FetchBodyStructure)
	}

	if len(fields.Headers) > 0 {
		section := &imap.BodySectionName{
			BodyPartName: imap.BodyPartName{
				Specifier: "HEADER.FIELDS",
				Fields:    fields.Headers,
			},
		}
		items = append(items, section.FetchItem())
	}

	if fields.Body {
		bodySection := &imap.BodySectionName{}
		items = append(items, bodySection.FetchItem())
	}

	return items
}

// CountMessagesBySender counts the messages from a given sender with improved performance.
func CountMessagesBySender(folder string, threshold int) ([][]string, error) {
	// Only fetch the fields we need for counting by sender
	fields := MessageFields{
		Envelope: true,
		Headers:  []string{"From"},
		BodyPeek: true,
	}

	messages, err := FetchMessages(folder, fields)
	if err != nil {
		return nil, fmt.Errorf("error fetching messages from folder %s: %w", folder, err)
	}

	// Pre-allocate map with estimated capacity
	senderCounts := make(map[string]int, len(messages)/2) // Assuming average of 2 messages per sender

	// Use a sync.Pool for string builders to reduce allocations
	pool := sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}

	// Process messages in parallel using worker pool
	const numWorkers = 4
	messageChan := make(chan *imap.Message, numWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex // Protects senderCounts

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for msg := range messageChan {
				if len(msg.Envelope.From) == 0 {
					continue
				}

				sb := pool.Get().(*strings.Builder)
				sb.Reset()
				sender := FormatAddress(msg.Envelope.From[0]) // <-- Capture the return value
				pool.Put(sb)

				if sender != "" {
					mu.Lock()
					senderCounts[sender]++
					mu.Unlock()
				}
			}
		}()
	}
	// Send messages to workers
	for _, msg := range messages {
		messageChan <- msg
	}
	close(messageChan)
	wg.Wait()

	// Pre-allocate slice with exact capacity
	senderCountList := make([]SenderCount, 0, len(senderCounts))
	for sender, count := range senderCounts {
		if count >= threshold {
			senderCountList = append(senderCountList, SenderCount{sender, count})
		}
	}

	// Sort in parallel for large datasets
	if len(senderCountList) > 1000 {
		parallelSort(senderCountList)
	} else {
		sort.Slice(senderCountList, func(i, j int) bool {
			return senderCountList[i].MessageCount > senderCountList[j].MessageCount
		})
	}

	// Pre-allocate result slice
	tableData := make([][]string, 0, len(senderCountList)+1)
	tableData = append(tableData, []string{"Sender", "Number of Messages"})

	for _, senderCount := range senderCountList {
		tableData = append(tableData, []string{
			senderCount.Sender,
			strconv.Itoa(senderCount.MessageCount), // Use strconv.Itoa instead of fmt.Sprintf
		})
	}

	return tableData, nil
}

// Helper function for parallel sorting of large datasets
func parallelSort(data []SenderCount) {
	if len(data) <= 1 {
		return
	}

	mid := len(data) / 2
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		parallelSort(data[:mid])
	}()
	parallelSort(data[mid:])
	wg.Wait()

	// Merge the sorted halves
	merged := make([]SenderCount, len(data))
	i, j, k := 0, mid, 0
	for i < mid && j < len(data) {
		if data[i].MessageCount > data[j].MessageCount {
			merged[k] = data[i]
			i++
		} else {
			merged[k] = data[j]
			j++
		}
		k++
	}
	copy(merged[k:], data[i:mid])
	copy(merged[k+mid-i:], data[j:])
	copy(data, merged)
}

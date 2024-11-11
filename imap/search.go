package imap

import (
	"encoding/json"
	"fmt"
	"github.com/emersion/go-imap"
	"time"
)

// SearchOptions represents the optional search parameters
type SearchOptions struct {
	To        *string    // Optional To address
	From      *string    // Optional From address
	Subject   *string    // Optional Subject
	StartDate *time.Time // Optional start date
	EndDate   *time.Time // Optional end date
	Seen      *bool      // Optional seen flag
}

func (opts SearchOptions) Serialize() string {
	jsonBytes, err := json.MarshalIndent(opts, "", "  ")
	if err != nil {
		log.Fatal().Msg("failed to serialize search options to JSON")
	}
	return string(jsonBytes)
}

func SearchMessages(account Account, mailbox string, criteria *imap.SearchCriteria) ([]*imap.Message, error) {
	// Connect to the server
	imapClient, err := GetImapClient(account)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer imapClient.Logout()

	if _, err := imapClient.Select(mailbox, true); err != nil {
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

	var seqNums []uint32

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

	log.Debug().Msgf("Final search criteria: %+v", serializeCriteria(criteria))

	return criteria
}

func serializeCriteria(criteria *imap.SearchCriteria) string {
	jsonBytes, err := json.MarshalIndent(criteria, "", "  ")
	if err != nil {
		log.Fatal().Msg("failed to serialize search options to JSON")
	}
	return string(jsonBytes)
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

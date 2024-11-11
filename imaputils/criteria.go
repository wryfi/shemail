package imaputils

import (
	"encoding/json"
	"github.com/emersion/go-imap"
)

// BuildSearchCriteria builds search criteria based on the given search options.
func BuildSearchCriteria(opts SearchOptions) *imap.SearchCriteria {
	criteria := initializeCriteria()

	addHeaderCriteria(criteria, opts)
	addDateCriteria(criteria, opts)
	addFlagCriteria(criteria, opts)

	logFinalCriteria(criteria)
	return criteria
}

// serializeCriteria serializes search criteria to json
func serializeCriteria(criteria *imap.SearchCriteria) string {
	jsonBytes, err := json.MarshalIndent(criteria, "", "  ")
	if err != nil {
		log.Fatal().Msg("failed to serialize search options to JSON")
	}
	return string(jsonBytes)
}

// initializeCriteria creates and initializes a new SearchCriteria struct
func initializeCriteria() *imap.SearchCriteria {
	return &imap.SearchCriteria{
		Header: make(map[string][]string),
	}
}

// addHeaderCriteria adds To, From, and Subject criteria if specified
func addHeaderCriteria(criteria *imap.SearchCriteria, opts SearchOptions) {
	headerFields := map[string]*string{
		"To":      opts.To,
		"From":    opts.From,
		"Subject": opts.Subject,
	}

	for field, value := range headerFields {
		if value != nil {
			criteria.Header[field] = []string{*value}
			log.Debug().Msgf("Adding %s criterion: %s", field, *value)
		}
	}
}

// addDateCriteria adds date-related search criteria
func addDateCriteria(criteria *imap.SearchCriteria, opts SearchOptions) {
	if opts.StartDate != nil {
		criteria.Since = *opts.StartDate
		criteria.SentSince = *opts.StartDate
		log.Debug().Msgf("Adding start date criteria: %s", opts.StartDate.String())
	}

	if opts.EndDate != nil {
		endDatePlusOne := opts.EndDate.AddDate(0, 0, 1)
		criteria.Before = endDatePlusOne
		criteria.SentBefore = endDatePlusOne
		log.Debug().Msgf("Adding end date criteria: %s", opts.EndDate.String())
	}
}

// addFlagCriteria adds seen/unseen flag criteria
func addFlagCriteria(criteria *imap.SearchCriteria, opts SearchOptions) {
	if opts.Seen != nil && *opts.Seen {
		criteria.WithFlags = []string{imap.SeenFlag}
		log.Debug().Msgf("Adding Seen criterion")
	}

	if opts.Unseen != nil && *opts.Unseen {
		criteria.WithoutFlags = []string{imap.SeenFlag}
		log.Debug().Msgf("Adding Unseen criterion")
	}
}

// logFinalCriteria logs the final search criteria for debugging
func logFinalCriteria(criteria *imap.SearchCriteria) {
	log.Debug().Msgf("Final search criteria: %+v", serializeCriteria(criteria))
}

// BuildORSearchCriteria creates an IMAP search criteria based on provided options using OR logic
func BuildORSearchCriteria(opts SearchOptions) *imap.SearchCriteria {
	criteriaList := buildIndividualCriteria(opts)
	combinedCriteria := combineCriteriaWithOR(criteriaList)
	logFinalCriteria(combinedCriteria)
	return combinedCriteria
}

// buildIndividualCriteria creates separate search criteria for each specified option
func buildIndividualCriteria(opts SearchOptions) []*imap.SearchCriteria {
	var criteriaList []*imap.SearchCriteria

	criteriaList = append(criteriaList, buildHeaderCriteria(opts)...)
	criteriaList = append(criteriaList, buildDateRangeCriteria(opts)...)
	criteriaList = append(criteriaList, buildFlagCriteria(opts)...)

	return criteriaList
}

// buildHeaderCriteria creates individual criteria for header fields
func buildHeaderCriteria(opts SearchOptions) []*imap.SearchCriteria {
	var criteria []*imap.SearchCriteria

	headerFields := map[string]*string{
		"To":      opts.To,
		"From":    opts.From,
		"Subject": opts.Subject,
	}

	for field, value := range headerFields {
		if value != nil {
			c := &imap.SearchCriteria{
				Header: map[string][]string{
					field: {*value},
				},
			}
			criteria = append(criteria, c)
		}
	}

	return criteria
}

// buildDateRangeCriteria creates criteria for date ranges
func buildDateRangeCriteria(opts SearchOptions) []*imap.SearchCriteria {
	var criteria []*imap.SearchCriteria

	// Handle start and end date together if both are present
	if opts.StartDate != nil && opts.EndDate != nil {
		endDatePlusOne := opts.EndDate.AddDate(0, 0, 1)
		c := &imap.SearchCriteria{
			Since:      *opts.StartDate,
			Before:     endDatePlusOne,
			SentSince:  *opts.StartDate,
			SentBefore: endDatePlusOne,
		}
		criteria = append(criteria, c)
		return criteria
	}

	// Handle individual dates if only one is present
	if opts.StartDate != nil {
		c := &imap.SearchCriteria{
			Since:     *opts.StartDate,
			SentSince: *opts.StartDate,
		}
		criteria = append(criteria, c)
	}

	if opts.EndDate != nil {
		endDatePlusOne := opts.EndDate.AddDate(0, 0, 1)
		c := &imap.SearchCriteria{
			Before:     endDatePlusOne,
			SentBefore: endDatePlusOne,
		}
		criteria = append(criteria, c)
	}

	return criteria
}

// buildFlagCriteria creates criteria for seen/unseen flags
func buildFlagCriteria(opts SearchOptions) []*imap.SearchCriteria {
	var criteria []*imap.SearchCriteria

	if opts.Seen != nil && *opts.Seen {
		c := &imap.SearchCriteria{
			WithFlags: []string{imap.SeenFlag},
		}
		criteria = append(criteria, c)
	}

	if opts.Unseen != nil && *opts.Unseen {
		c := &imap.SearchCriteria{
			WithoutFlags: []string{imap.SeenFlag},
		}
		criteria = append(criteria, c)
	}

	return criteria
}

// combineCriteriaWithOR combines multiple search criteria using OR logic
func combineCriteriaWithOR(criteriaList []*imap.SearchCriteria) *imap.SearchCriteria {
	switch len(criteriaList) {
	case 0:
		return &imap.SearchCriteria{}
	case 1:
		return criteriaList[0]
	default:
		return buildORChain(criteriaList)
	}
}

// buildORChain creates a chain of OR conditions from the criteria list
func buildORChain(criteriaList []*imap.SearchCriteria) *imap.SearchCriteria {
	if len(criteriaList) == 2 {
		return &imap.SearchCriteria{
			Or: [][2]*imap.SearchCriteria{{criteriaList[0], criteriaList[1]}},
		}
	}

	// For more than 2 criteria, chain them: (A OR B) OR C
	result := &imap.SearchCriteria{
		Or: [][2]*imap.SearchCriteria{
			{
				criteriaList[0],
				buildORChain(criteriaList[1:]),
			},
		},
	}

	return result
}

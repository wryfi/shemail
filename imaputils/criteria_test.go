package imaputils

import (
	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestBuildSearchCriteria(t *testing.T) {
	// Helper function to create string pointer
	strPtr := func(s string) *string {
		return &s
	}

	// Helper function to create bool pointer
	boolPtr := func(b bool) *bool {
		return &b
	}

	// Helper function to create time.Time pointer
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	// Helper function to create uint32 pointer
	uint32Ptr := func(value uint32) *uint32 {
		return &value
	}

	tests := []struct {
		name     string
		opts     SearchOptions
		expected *imap.SearchCriteria
	}{
		{
			name: "Empty options",
			opts: SearchOptions{},
			expected: &imap.SearchCriteria{
				Header: make(map[string][]string),
			},
		},
		{
			name: "Header criteria only",
			opts: SearchOptions{
				To:   strPtr("recipient@example.com"),
				From: strPtr("sender@example.com"),
			},
			expected: &imap.SearchCriteria{
				Header: map[string][]string{
					"To":   {"recipient@example.com"},
					"From": {"sender@example.com"},
				},
			},
		},
		{
			name: "Date criteria only",
			opts: SearchOptions{
				StartDate: timePtr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
				EndDate:   timePtr(time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)),
			},
			expected: &imap.SearchCriteria{
				Header: make(map[string][]string),
				Since:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Before: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Flag criteria only",
			opts: SearchOptions{
				Seen:   boolPtr(true),
				Unseen: boolPtr(false),
			},
			expected: &imap.SearchCriteria{
				Header:    make(map[string][]string),
				WithFlags: []string{imap.SeenFlag},
			},
		},
		{
			name: "Negated header criteria",
			opts: SearchOptions{
				From:    strPtr("company@example.com"),
				NotFrom: strPtr("noreply@example.com"),
			},
			expected: &imap.SearchCriteria{
				Header: map[string][]string{
					"From": {"company@example.com"},
				},
				Not: []*imap.SearchCriteria{
					{Header: map[string][]string{"From": {"noreply@example.com"}}},
				},
			},
		},
		{
			name: "Size criteria only",
			opts: SearchOptions{
				LargerThan:  uint32Ptr(10 * 1024 * 1024),
				SmallerThan: uint32Ptr(1024 * 1024),
			},
			expected: &imap.SearchCriteria{
				Header:  make(map[string][]string),
				Larger:  10 * 1024 * 1024,
				Smaller: 1024 * 1024,
			},
		},
		{
			name: "Combined criteria",
			opts: SearchOptions{
				To:        strPtr("recipient@example.com"),
				StartDate: timePtr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
				Seen:      boolPtr(true),
			},
			expected: &imap.SearchCriteria{
				Header: map[string][]string{
					"To": {"recipient@example.com"},
				},
				Since:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				WithFlags: []string{imap.SeenFlag},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildSearchCriteria(tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildORSearchCriteria(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	boolPtr := func(b bool) *bool { return &b }

	tests := []struct {
		name   string
		opts   SearchOptions
		verify func(t *testing.T, result *imap.SearchCriteria)
	}{
		{
			name: "Single criterion",
			opts: SearchOptions{
				From: strPtr("sender@example.com"),
			},
			verify: func(t *testing.T, result *imap.SearchCriteria) {
				assert.Len(t, result.Header, 1, "Should have one header")
				values := result.Header["From"]
				assert.Equal(t, []string{"sender@example.com"}, values, "From should match")
			},
		},
		{
			name: "Two criteria",
			opts: SearchOptions{
				To:   strPtr("recipient@example.com"),
				From: strPtr("sender@example.com"),
			},
			verify: func(t *testing.T, result *imap.SearchCriteria) {
				// Verify it's an OR condition
				assert.Len(t, result.Or, 1)
				assert.Len(t, result.Or[0], 2)

				// Verify both criteria exist somewhere in the OR
				criteria := []*imap.SearchCriteria{result.Or[0][0], result.Or[0][1]}
				foundTo := false
				foundFrom := false

				for _, c := range criteria {
					if c.Header.Get("To") == "recipient@example.com" {
						foundTo = true
					}
					if c.Header.Get("From") == "sender@example.com" {
						foundFrom = true
					}
				}

				assert.True(t, foundTo, "Should find To criterion")
				assert.True(t, foundFrom, "Should find From criterion")
			},
		},
		{
			name: "Three criteria",
			opts: SearchOptions{
				To:   strPtr("recipient@example.com"),
				From: strPtr("sender@example.com"),
				Seen: boolPtr(true),
			},
			verify: func(t *testing.T, result *imap.SearchCriteria) {
				// Collect the leaf criteria from the OR tree.
				var leaves []*imap.SearchCriteria
				var walk func(c *imap.SearchCriteria)
				walk = func(c *imap.SearchCriteria) {
					if len(c.Or) == 1 {
						walk(c.Or[0][0])
						walk(c.Or[0][1])
						return
					}
					leaves = append(leaves, c)
				}
				walk(result)

				foundTo := false
				foundFrom := false
				foundFlag := false
				for _, c := range leaves {
					if c.Header.Get("To") == "recipient@example.com" {
						foundTo = true
					}
					if c.Header.Get("From") == "sender@example.com" {
						foundFrom = true
					}
					if len(c.WithFlags) > 0 {
						foundFlag = true
					}
				}

				assert.True(t, foundTo, "Should find To criterion")
				assert.True(t, foundFrom, "Should find From criterion")
				assert.True(t, foundFlag, "Should find flag criterion")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildORSearchCriteria(tt.opts)
			tt.verify(t, result)
		})
	}
}

func TestBuildIndividualCriteria(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	timePtr := func(t time.Time) *time.Time { return &t }
	boolPtr := func(b bool) *bool { return &b }

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	opts := SearchOptions{
		To:        strPtr("recipient@example.com"),
		From:      strPtr("sender@example.com"),
		StartDate: timePtr(startDate),
		EndDate:   timePtr(endDate),
		Seen:      boolPtr(true),
	}

	result := buildIndividualCriteria(opts)

	// We expect 4 criteria: To, From, DateRange (combined), and Seen
	assert.Equal(t, 4, len(result))

	// Test that each type of criteria is present
	hasHeader := false
	hasDateRange := false
	hasFlag := false

	for _, criteria := range result {
		if len(criteria.Header) > 0 {
			hasHeader = true
		}
		if !criteria.Since.IsZero() || !criteria.Before.IsZero() {
			hasDateRange = true
		}
		if len(criteria.WithFlags) > 0 {
			hasFlag = true
		}
	}

	assert.True(t, hasHeader, "Should have header criteria")
	assert.True(t, hasDateRange, "Should have date range criteria")
	assert.True(t, hasFlag, "Should have flag criteria")
}

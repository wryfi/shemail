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
				To:      strPtr("recipient@example.com"),
				From:    strPtr("sender@example.com"),
				Subject: strPtr("test subject"),
			},
			expected: &imap.SearchCriteria{
				Header: map[string][]string{
					"To":      {"recipient@example.com"},
					"From":    {"sender@example.com"},
					"Subject": {"test subject"},
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
				Header:     make(map[string][]string),
				Since:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Before:     time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				SentSince:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				SentBefore: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
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
				SentSince: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
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

	tests := []struct {
		name     string
		opts     SearchOptions
		expected *imap.SearchCriteria
	}{
		{
			name: "Single criterion",
			opts: SearchOptions{
				Subject: strPtr("test"),
			},
			expected: &imap.SearchCriteria{
				Header: map[string][]string{
					"Subject": {"test"},
				},
			},
		},
		{
			name: "Two criteria",
			opts: SearchOptions{
				To:   strPtr("recipient@example.com"),
				From: strPtr("sender@example.com"),
			},
			expected: &imap.SearchCriteria{
				Or: [][2]*imap.SearchCriteria{{
					{
						Header: map[string][]string{"To": {"recipient@example.com"}},
					},
					{
						Header: map[string][]string{"From": {"sender@example.com"}},
					},
				}},
			},
		},
		{
			name: "Three criteria",
			opts: SearchOptions{
				To:      strPtr("recipient@example.com"),
				From:    strPtr("sender@example.com"),
				Subject: strPtr("test"),
			},
			expected: &imap.SearchCriteria{
				Or: [][2]*imap.SearchCriteria{{
					{
						Header: map[string][]string{"To": {"recipient@example.com"}},
					},
					{
						Or: [][2]*imap.SearchCriteria{{
							{
								Header: map[string][]string{"From": {"sender@example.com"}},
							},
							{
								Header: map[string][]string{"Subject": {"test"}},
							},
						}},
					},
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildORSearchCriteria(tt.opts)
			assert.Equal(t, tt.expected, result)
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
		Subject:   strPtr("test"),
		StartDate: timePtr(startDate),
		EndDate:   timePtr(endDate),
		Seen:      boolPtr(true),
	}

	result := buildIndividualCriteria(opts)

	// We expect 5 criteria: To, From, Subject, DateRange (combined), and Seen
	assert.Equal(t, 5, len(result))

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

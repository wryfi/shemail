package util

import (
	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		expected  string
	}{
		{
			name:      "string shorter than max length",
			input:     "hello",
			maxLength: 10,
			expected:  "hello",
		},
		{
			name:      "string equal to max length",
			input:     "hello",
			maxLength: 5,
			expected:  "hello",
		},
		{
			name:      "string longer than max length",
			input:     "hello world",
			maxLength: 8,
			expected:  "hello...",
		},
		{
			name:      "very short max length",
			input:     "hello",
			maxLength: 2,
			expected:  "he",
		},
		{
			name:      "unicode string",
			input:     "hello世界",
			maxLength: 6,
			expected:  "hel...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLength)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStringPtr(t *testing.T) {
	input := "test"
	ptr := StringPtr(input)
	assert.NotNil(t, ptr)
	assert.Equal(t, input, *ptr)
}

func TestTimePtr(t *testing.T) {
	now := time.Now()
	ptr := TimePtr(now)
	assert.NotNil(t, ptr)
	assert.Equal(t, now, *ptr)
}

func TestBoolPtr(t *testing.T) {
	tests := []bool{true, false}
	for _, v := range tests {
		ptr := BoolPtr(v)
		assert.NotNil(t, ptr)
		assert.Equal(t, v, *ptr)
	}
}

func TestDateFromString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    time.Time
		expectError bool
	}{
		{
			name:        "valid date",
			input:       "2024-01-01",
			expected:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expectError: false,
		},
		{
			name:        "invalid format",
			input:       "01-01-2024",
			expectError: true,
		},
		{
			name:        "invalid date",
			input:       "2024-13-01",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DateFromString(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestMessageDate(t *testing.T) {
	// Create a fixed timezone for testing
	nyc, err := time.LoadLocation("America/New_York")
	assert.NoError(t, err)

	// Test date in different timezone
	originalDate := time.Date(2024, 1, 1, 12, 0, 0, 0, nyc)

	md := NewMessageDate(originalDate)

	// Test Original field
	assert.Equal(t, originalDate, md.Original)

	// Test Normalized field (should be in UTC)
	assert.Equal(t, time.UTC, md.Normalized.Location())

	// Test LocalizeToZone
	localizedDate := md.LocalizeToZone(nyc)
	assert.Equal(t, nyc, localizedDate.Location())

	// Test FormatConsistent
	formatted := md.FormatConsistent(nyc)
	assert.Contains(t, formatted, "2024-01-01")
	assert.Contains(t, formatted, "EST") // Should contain timezone abbreviation
}

func TestParseSize(t *testing.T) {
	valid := map[string]uint32{
		"1024": 1024,
		"500":  500,
		"1K":   1024,
		"1KB":  1024,
		"10M":  10 * 1024 * 1024,
		"10mb": 10 * 1024 * 1024,
		"1.5M": 1572864,
		"2G":   2 * 1024 * 1024 * 1024,
		" 4K ": 4096,
	}
	for input, want := range valid {
		got, err := ParseSize(input)
		if err != nil {
			t.Errorf("ParseSize(%q) unexpected error: %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("ParseSize(%q) = %d, want %d", input, got, want)
		}
	}

	invalid := []string{"", "abc", "10X", "-5M", "5G"} // 5G overflows uint32
	for _, input := range invalid {
		if _, err := ParseSize(input); err == nil {
			t.Errorf("ParseSize(%q) expected an error, got none", input)
		}
	}
}

func TestFormatSize(t *testing.T) {
	cases := map[uint32]string{
		0:                  "0B",
		512:                "512B",
		1024:               "1.0K",
		1536:               "1.5K",
		1024 * 1024:        "1.0M",
		10 * 1024 * 1024:   "10.0M",
		1024 * 1024 * 1024: "1.0G",
	}
	for size, want := range cases {
		if got := FormatSize(size); got != want {
			t.Errorf("FormatSize(%d) = %q, want %q", size, got, want)
		}
	}
}

func TestTabulateMessages(t *testing.T) {
	messages := []*imap.Message{
		{
			Envelope: &imap.Envelope{
				Subject: "Test Subject",
				From:    []*imap.Address{{PersonalName: "John Doe", MailboxName: "john", HostName: "example.com"}},
				To:      []*imap.Address{{PersonalName: "Jane Doe", MailboxName: "jane", HostName: "example.com"}},
			},
			InternalDate: time.Now(),
		},
	}

	table, err := TabulateMessages(messages)
	assert.NoError(t, err)
	assert.NotNil(t, table)
}

func TestTabulateSenders(t *testing.T) {
	data := [][]string{
		{"Header1", "Header2"},
		{"Value1", "Value2"},
	}

	table := TabulateSenders(data)
	assert.NotNil(t, table)
}

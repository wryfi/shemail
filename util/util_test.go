package util

import (
	"github.com/emersion/go-imap"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/wryfi/shemail/imaputils"
	"strings"
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

func TestUnreadMarker(t *testing.T) {
	assert.Equal(t, "", UnreadMarker([]string{imap.SeenFlag}), "seen message has no marker")
	assert.Equal(t, "", UnreadMarker([]string{"\\Answered", imap.SeenFlag}), "seen among others")
	assert.Equal(t, "●", UnreadMarker(nil), "no flags = unread")
	assert.Equal(t, "●", UnreadMarker([]string{"\\Flagged"}), "flagged but not seen = unread")
}

func TestIsUnread(t *testing.T) {
	assert.False(t, IsUnread([]string{imap.SeenFlag}), "seen message is read")
	assert.False(t, IsUnread([]string{"\\Answered", imap.SeenFlag}), "seen among others is read")
	assert.True(t, IsUnread(nil), "no flags = unread")
	assert.True(t, IsUnread([]string{"\\Flagged"}), "flagged but not seen = unread")
}

func TestFormatMessageRows(t *testing.T) {
	viper.Set("timezone", "UTC")
	defer viper.Set("timezone", "")

	fixedDate := time.Date(2026, 1, 26, 15, 8, 17, 0, time.UTC)

	t.Run("formats cells in column order and flags read state", func(t *testing.T) {
		messages := []*imap.Message{
			{
				Uid:          54099,
				InternalDate: fixedDate,
				Size:         1536, // 1.5K
				Envelope: &imap.Envelope{
					Subject: "Renewal Notice",
					From:    []*imap.Address{{MailboxName: "noreply", HostName: "cloudflare.com"}},
					To:      []*imap.Address{{MailboxName: "ch", HostName: "wryfi.net"}},
				},
				Flags: []string{imap.SeenFlag},
			},
		}

		rows, err := FormatMessageRows(messages)
		assert.NoError(t, err)
		assert.Len(t, rows, 1)

		row := rows[0]
		// No leading unread/checkbox column: cells align 1:1 with MessageColumns.
		assert.Len(t, row.Cells, len(MessageColumns))
		assert.Equal(t, "2026-01-26 15:08:17 +0000 UTC", row.Cells[0])
		assert.Equal(t, "1.5K", row.Cells[1])
		assert.Equal(t, "noreply@cloudflare.com", row.Cells[2])
		assert.Equal(t, "ch@wryfi.net", row.Cells[3])
		assert.Equal(t, "Renewal Notice", row.Cells[4])
		assert.False(t, row.Unread, "seen message is not unread")
	})

	t.Run("unread when Seen flag absent", func(t *testing.T) {
		messages := []*imap.Message{
			{Envelope: &imap.Envelope{Subject: "hi"}, Flags: []string{"\\Flagged"}},
			{Envelope: &imap.Envelope{Subject: "yo"}, Flags: []string{imap.SeenFlag}},
		}
		rows, err := FormatMessageRows(messages)
		assert.NoError(t, err)
		assert.True(t, rows[0].Unread, "no Seen flag = unread")
		assert.False(t, rows[1].Unread, "Seen flag = read")
	})

	t.Run("nil envelope falls back to placeholders", func(t *testing.T) {
		messages := []*imap.Message{
			{Uid: 1, InternalDate: fixedDate, Size: 1536, Envelope: nil},
		}
		rows, err := FormatMessageRows(messages)
		assert.NoError(t, err)
		assert.Len(t, rows, 1)
		assert.Equal(t,
			[]string{"2026-01-26 15:08:17 +0000 UTC", "1.5K", "(unknown)", "(unknown)", "(unknown)"},
			rows[0].Cells,
		)
		assert.True(t, rows[0].Unread, "no flags = unread")
	})

	t.Run("empty subject shows placeholder", func(t *testing.T) {
		messages := []*imap.Message{{Envelope: &imap.Envelope{Subject: ""}}}
		rows, err := FormatMessageRows(messages)
		assert.NoError(t, err)
		assert.Equal(t, "(unknown)", rows[0].Cells[4])
	})

	t.Run("truncates from to 30 and subject to 60 with ellipsis", func(t *testing.T) {
		messages := []*imap.Message{
			{
				Envelope: &imap.Envelope{
					Subject: strings.Repeat("x", 80),
					From: []*imap.Address{
						{MailboxName: "noreply-very-long-address-name", HostName: "notify.cloudflare.com"},
					},
				},
			},
		}
		rows, err := FormatMessageRows(messages)
		assert.NoError(t, err)
		from := rows[0].Cells[2]
		subject := rows[0].Cells[4]
		assert.Len(t, []rune(from), 30)
		assert.True(t, strings.HasSuffix(from, "..."), "long from is ellipsized")
		assert.Len(t, []rune(subject), 60)
		assert.True(t, strings.HasSuffix(subject, "..."), "long subject is ellipsized")
	})
}

func TestRenderMessages(t *testing.T) {
	viper.Set("timezone", "UTC")
	defer viper.Set("timezone", "")

	messages := []*imap.Message{
		{
			Envelope: &imap.Envelope{
				Subject: "Test Subject",
				From:    []*imap.Address{{PersonalName: "John Doe", MailboxName: "john", HostName: "example.com"}},
				To:      []*imap.Address{{PersonalName: "Jane Doe", MailboxName: "jane", HostName: "example.com"}},
			},
			InternalDate: time.Date(2026, 1, 26, 15, 8, 17, 0, time.UTC),
			Flags:        []string{imap.SeenFlag},
		},
	}

	rendered, err := RenderMessages(messages)
	assert.NoError(t, err)
	assert.Contains(t, rendered, "Subject", "includes column header")
	assert.Contains(t, rendered, "Test Subject", "includes the subject cell")
	assert.Contains(t, rendered, "john@example.com", "includes the from cell")
	assert.Contains(t, rendered, "Found 1 messages", "includes the count caption")
}

func TestRenderSenders(t *testing.T) {
	data := [][]string{
		{"Sender", "Number of Messages"},
		{"noreply@example.com", "42"},
	}

	rendered := RenderSenders(data)
	assert.Contains(t, rendered, "Sender", "includes header")
	assert.Contains(t, rendered, "noreply@example.com", "includes the sender cell")
	assert.Contains(t, rendered, "42", "includes the count cell")

	assert.Empty(t, RenderSenders(nil), "no data renders nothing")
}

func TestRenderFolders(t *testing.T) {
	folders := []imaputils.FolderStatus{
		{Name: "INBOX", Selectable: true, Messages: 128, Unseen: 3},
		{Name: "Archive", Selectable: false}, // container folder shows "-"
	}

	rendered := RenderFolders(folders, false)
	assert.Contains(t, rendered, "Folder", "includes header")
	assert.Contains(t, rendered, "INBOX")
	assert.Contains(t, rendered, "128")
	assert.Contains(t, rendered, "-", "non-selectable folder shows a dash for counts")
}

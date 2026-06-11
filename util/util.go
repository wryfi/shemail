package util

import (
	"bufio"
	"fmt"
	"github.com/charmbracelet/lipgloss"
	ltable "github.com/charmbracelet/lipgloss/table"
	"github.com/emersion/go-imap"
	"github.com/spf13/viper"
	"github.com/wryfi/shemail/imaputils"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// TruncateString truncates a string to the specified length and appends an ellipsis if needed.
func TruncateString(str string, maxLength int) string {
	if utf8.RuneCountInString(str) <= maxLength {
		return str
	}

	// Truncate the string and add ellipsis
	runes := []rune(str)
	if maxLength > 3 {
		return string(runes[:maxLength-3]) + "..."
	}
	return string(runes[:maxLength])
}

// Helper function to create a string pointer
func StringPtr(s string) *string {
	return &s
}

// Helper function to create a time pointer
func TimePtr(t time.Time) *time.Time {
	return &t
}

// Helper function to create a bool pointer
func BoolPtr(b bool) *bool {
	return &b
}

// Helper function to create a uint32 pointer
func Uint32Ptr(value uint32) *uint32 {
	return &value
}

// ParseSize parses a human-friendly size string into a number of bytes. It
// accepts a bare byte count ("1024") or a value with a binary unit suffix
// (K/KB, M/MB, G/GB, case-insensitive), e.g. "10M" = 10 MiB. The result must
// fit in a uint32, which is what IMAP SEARCH LARGER/SMALLER use.
func ParseSize(text string) (uint32, error) {
	trimmed := strings.TrimSpace(strings.ToUpper(text))
	if trimmed == "" {
		return 0, fmt.Errorf("empty size")
	}
	trimmed = strings.TrimSuffix(trimmed, "B")

	var multiplier float64 = 1
	switch {
	case strings.HasSuffix(trimmed, "K"):
		multiplier = 1024
		trimmed = strings.TrimSuffix(trimmed, "K")
	case strings.HasSuffix(trimmed, "M"):
		multiplier = 1024 * 1024
		trimmed = strings.TrimSuffix(trimmed, "M")
	case strings.HasSuffix(trimmed, "G"):
		multiplier = 1024 * 1024 * 1024
		trimmed = strings.TrimSuffix(trimmed, "G")
	}

	value, err := strconv.ParseFloat(strings.TrimSpace(trimmed), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q", text)
	}
	if value < 0 {
		return 0, fmt.Errorf("size cannot be negative: %q", text)
	}

	bytes := value * multiplier
	if bytes > float64(^uint32(0)) {
		return 0, fmt.Errorf("size %q exceeds the maximum of 4 GiB", text)
	}
	return uint32(bytes), nil
}

// FormatSize renders a byte count as a short human-readable string using binary
// units, e.g. 1572864 -> "1.5M".
func FormatSize(size uint32) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%dB", size)
	}
	value := float64(size)
	units := []string{"K", "M", "G"}
	unitIndex := -1
	for value >= unit && unitIndex < len(units)-1 {
		value /= unit
		unitIndex++
	}
	return fmt.Sprintf("%.1f%s", value, units[unitIndex])
}

// DateFromString takes a date string of format "yyyy-mm-dd" and returns a Time
func DateFromString(dateStr string) (time.Time, error) {
	layout := "2006-01-02"
	date, err := time.Parse(layout, dateStr)
	if err != nil {
		return time.Time{}, err
	}
	return date, nil
}

// MessageDate represents a normalized message date with timezone handling
type MessageDate struct {
	Original   time.Time
	Normalized time.Time
}

// NewMessageDate creates a new MessageDate from an IMAP internal date
func NewMessageDate(internalDate time.Time) MessageDate {
	// First convert to UTC to normalize
	utcDate := internalDate.UTC()

	return MessageDate{
		Original:   internalDate,
		Normalized: utcDate,
	}
}

// LocalizeToZone returns the date in the specified timezone
func (md MessageDate) LocalizeToZone(timezone *time.Location) time.Time {
	return md.Normalized.In(timezone)
}

// FormatConsistent returns a consistently formatted date string in the specified timezone
func (md MessageDate) FormatConsistent(timezone *time.Location) string {
	localTime := md.LocalizeToZone(timezone)
	return localTime.Format("2006-01-02 15:04:05 -0700 MST")
}

// IsUnread reports whether a message lacks the \Seen flag.
func IsUnread(flags []string) bool {
	for _, flag := range flags {
		if flag == imap.SeenFlag {
			return false
		}
	}
	return true
}

// UnreadMarker returns a dot for messages lacking the \Seen flag (unread) and
// an empty string for read messages, for use as a compact status indicator.
func UnreadMarker(flags []string) string {
	if IsUnread(flags) {
		return "●"
	}
	return ""
}

// MessageColumns are the message-table column headers, in order. The leading
// status column is intentionally omitted: unread state is carried on each
// MessageRow (rendered as bold), and the interactive picker prepends its own
// checkbox column. This is the single source of truth shared by the static
// table renderer and the picker.
var MessageColumns = []string{"Date", "Size", "From", "To", "Subject"}

const (
	dateColumnWidth    = 29 // "2006-01-02 15:04:05 -0700 MST"
	sizeColumnWidth    = 6
	fromColumnWidth    = 30
	toColumnWidth      = 30
	subjectColumnWidth = 60
)

// messageColumnWidths are the display widths for MessageColumns, in order. The
// picker fixes its columns to these so they don't jump as rows scroll; the
// static renderer auto-sizes within them since cells are pre-truncated here.
var messageColumnWidths = []int{dateColumnWidth, sizeColumnWidth, fromColumnWidth, toColumnWidth, subjectColumnWidth}

// MessageRow is one formatted message: its cells (aligned 1:1 with
// MessageColumns) and whether the message is unread, so renderers can style
// unread rows rather than carrying a separate marker column.
type MessageRow struct {
	Cells  []string
	Unread bool
}

// FormatMessageRows formats messages into display rows, the single source of
// truth for how a message renders. Cells are pre-truncated to their column
// widths. A message without an envelope (malformed or partial fetch) falls back
// to placeholders rather than panicking.
func FormatMessageRows(messages []*imap.Message) ([]MessageRow, error) {
	tzString := viper.GetString("timezone")
	tz, err := time.LoadLocation(tzString)
	if err != nil {
		return nil, fmt.Errorf("error loading timezone %q: %w", tzString, err)
	}

	rows := make([]MessageRow, 0, len(messages))
	for _, message := range messages {
		unread := IsUnread(message.Flags)
		date := NewMessageDate(message.InternalDate).FormatConsistent(tz)
		size := FormatSize(message.Size)

		if message.Envelope == nil {
			rows = append(rows, MessageRow{
				Cells:  []string{date, size, "(unknown)", "(unknown)", "(unknown)"},
				Unread: unread,
			})
			continue
		}

		subject := "(unknown)"
		if message.Envelope.Subject != "" {
			subject = TruncateString(message.Envelope.Subject, subjectColumnWidth)
		}
		from := TruncateString(imaputils.FormatAddressesCSV(message.Envelope.From), fromColumnWidth)
		to := TruncateString(imaputils.FormatAddressesCSV(message.Envelope.To), toColumnWidth)
		rows = append(rows, MessageRow{
			Cells:  []string{date, size, from, to, subject},
			Unread: unread,
		})
	}
	return rows, nil
}

// Shared table styling, used by every tabular renderer (messages, folders,
// senders) and the interactive picker, so all of shemail's tables look alike.
var (
	tableBaseStyle  = lipgloss.NewStyle().Padding(0, 1)
	tableBoldStyle  = tableBaseStyle.Bold(true)  // header cells and unread rows
	tableMutedStyle = tableBaseStyle.Faint(true) // read rows, de-emphasized
)

// styledTable returns a lipgloss table with shemail's shared look: a single rule
// under the header, no outer box or column/row separators, and one space of
// horizontal padding per cell. The caller supplies the per-cell StyleFunc (which
// should return tableBoldStyle for the header row).
func styledTable(headers []string, styleFunc func(row, col int) lipgloss.Style) *ltable.Table {
	return ltable.New().
		Border(lipgloss.NormalBorder()).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		BorderRow(false).
		BorderHeader(true).
		Headers(headers...).
		StyleFunc(styleFunc)
}

// RenderMessages renders messages as a table string for static (non-interactive)
// display: the shared columns, with unread rows in bold (replacing the old dot
// column) and a trailing count caption. Borders are reduced to a single header
// rule so the static view matches the interactive picker, which draws the same
// shape plus a leading checkbox column.
func RenderMessages(messages []*imap.Message) (string, error) {
	rows, err := FormatMessageRows(messages)
	if err != nil {
		return "", err
	}

	table := styledTable(MessageColumns, func(row, col int) lipgloss.Style {
		// Header and unread rows are bold; read rows are faint, so the unread
		// messages stand out by contrast.
		if row == ltable.HeaderRow {
			return tableBoldStyle
		}
		if row >= 0 && row < len(rows) && rows[row].Unread {
			return tableBoldStyle
		}
		return tableMutedStyle
	})

	for _, row := range rows {
		table.Row(row.Cells...)
	}

	return fmt.Sprintf("%s\nFound %d messages", table.String(), len(messages)), nil
}

// RenderFolders renders a folder listing with message and unread counts as a
// table string in the shared style. When withDates is true it also includes the
// date range of each folder's messages. Non-selectable container folders (and
// empty folders, for dates) show "-". The numeric count columns are
// right-aligned.
func RenderFolders(folders []imaputils.FolderStatus, withDates bool) string {
	headers := []string{"Folder", "Messages", "Unread"}
	if withDates {
		headers = append(headers, "Oldest", "Newest")
	}

	// Messages (col 1) and Unread (col 2) are numeric; right-align them.
	numeric := func(col int) bool { return col == 1 || col == 2 }
	table := styledTable(headers, func(row, col int) lipgloss.Style {
		style := tableBaseStyle
		if row == ltable.HeaderRow {
			style = tableBoldStyle
		}
		if numeric(col) {
			style = style.Align(lipgloss.Right)
		}
		return style
	})

	for _, folder := range folders {
		messages, unread := "-", "-"
		if folder.Selectable {
			messages = strconv.Itoa(int(folder.Messages))
			unread = strconv.Itoa(int(folder.Unseen))
		}
		row := []string{folder.Name, messages, unread}
		if withDates {
			row = append(row, formatDate(folder.Oldest), formatDate(folder.Newest))
		}
		table.Row(row...)
	}

	return table.String()
}

// formatDate renders a date as YYYY-MM-DD, or "-" for the zero value.
func formatDate(date time.Time) string {
	if date.IsZero() {
		return "-"
	}
	return date.Format("2006-01-02")
}

// RenderSenders renders sender/count data (data[0] is the header row) as a table
// string in the shared style, with the trailing count column right-aligned.
func RenderSenders(data [][]string) string {
	if len(data) == 0 {
		return ""
	}

	headers := data[0]
	countColumn := len(headers) - 1 // the message-count column
	table := styledTable(headers, func(row, col int) lipgloss.Style {
		style := tableBaseStyle
		if row == ltable.HeaderRow {
			style = tableBoldStyle
		}
		if col == countColumn {
			style = style.Align(lipgloss.Right)
		}
		return style
	})

	for _, row := range data[1:] {
		table.Row(row...)
	}

	return table.String()
}

// GetConfirmation prompts the user for confirmation before proceeding
func GetConfirmation(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", prompt)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return false
		}

		response = strings.ToLower(strings.TrimSpace(response))

		switch response {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("Please answer with yes/no or y/n")
		}
	}
}

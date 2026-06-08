package util

import (
	"bufio"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/olekukonko/tablewriter"
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

// TabulateMessages takes a list of imap messages and displays them in a table
func TabulateMessages(messages []*imap.Message) (*tablewriter.Table, error) {
	tzString := viper.GetString("timezone")
	tz, err := time.LoadLocation(tzString)
	if err != nil {
		return &tablewriter.Table{}, fmt.Errorf("Error loading timezone: %s: %w", tzString, err)
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Date", "Size", "From", "To", "Subject"})
	table.SetBorder(false)
	//table.SetRowLine(true)
	table.SetAutoWrapText(false)
	table.SetCaption(true, fmt.Sprintf("Found %d messages", len(messages)))

	for _, message := range messages {
		msgDate := NewMessageDate(message.InternalDate)
		date := msgDate.FormatConsistent(tz)
		size := FormatSize(message.Size)

		// A message may arrive without an envelope (malformed message or a
		// partial fetch); fall back to placeholders rather than panicking.
		if message.Envelope == nil {
			table.Append([]string{date, size, "(unknown)", "(unknown)", "(unknown)"})
			continue
		}

		subject := "(unknown)"
		if message.Envelope.Subject != "" {
			subject = TruncateString(message.Envelope.Subject, 60)
		}
		from := TruncateString(imaputils.FormatAddressesCSV(message.Envelope.From), 30)
		to := imaputils.FormatAddressesCSV(message.Envelope.To)
		table.Append([]string{date, size, from, to, subject})

	}
	return table, nil
}

// TabulateFolders renders a list of folders with their message and unread
// counts. Non-selectable container folders show "-" for their counts.
func TabulateFolders(folders []imaputils.FolderStatus) *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Folder", "Messages", "Unread"})
	table.SetBorder(false)
	table.SetAutoWrapText(false)
	table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})

	for _, folder := range folders {
		messages := "-"
		unread := "-"
		if folder.Selectable {
			messages = strconv.Itoa(int(folder.Messages))
			unread = strconv.Itoa(int(folder.Unseen))
		}
		table.Append([]string{folder.Name, messages, unread})
	}

	return table
}

// TabulateSenders creates a table from the given data and renders it to the terminal
func TabulateSenders(data [][]string) *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(data[0])
	table.SetBorder(false)

	for _, v := range data[1:] {
		table.Append(v)
	}

	return table
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

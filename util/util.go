package util

import (
	"bufio"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/viper"
	"github.com/wryfi/shemail/imaputils"
	"os"
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
	table.SetHeader([]string{"Date", "From", "To", "Subject"})
	table.SetBorder(false)
	//table.SetRowLine(true)
	table.SetAutoWrapText(false)
	table.SetCaption(true, fmt.Sprintf("Found %d messages", len(messages)))

	for _, message := range messages {
		subject := "(unknown)"
		if message.Envelope.Subject != "" {
			subject = TruncateString(message.Envelope.Subject, 60)
		}
		msgDate := NewMessageDate(message.InternalDate)
		date := msgDate.FormatConsistent(tz)
		from := TruncateString(imaputils.FormatAddressesCSV(message.Envelope.From), 30)
		to := imaputils.FormatAddressesCSV(message.Envelope.To)
		table.Append([]string{date, from, to, subject})

	}
	return table, nil
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

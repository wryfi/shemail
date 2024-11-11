package util

import (
	"github.com/emersion/go-imap"
	"github.com/olekukonko/tablewriter"
	sheimap "github.com/wryfi/shemail/imap"
	"os"
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

func DateFromString(dateStr string) (time.Time, error) {
	layout := "2006-01-02"
	date, err := time.Parse(layout, dateStr)
	if err != nil {
		return time.Time{}, err
	}
	return date, nil
}

// TabulateMessages takes a list of imap messages and displays them in a table
func TabulateMessages(messages []*imap.Message) *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Date", "From", "To", "Subject"})
	table.SetBorder(false)
	//table.SetRowLine(true)
	table.SetAutoWrapText(false)

	for _, message := range messages {
		subject := "(unknown)"
		if message.Envelope.Subject != "" {
			subject = TruncateString(message.Envelope.Subject, 60)
		}
		date := message.InternalDate.String()
		from := TruncateString(sheimap.FormatAddressesCSV(message.Envelope.From), 30)
		to := sheimap.FormatAddressesCSV(message.Envelope.To)
		table.Append([]string{date, from, to, subject})

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

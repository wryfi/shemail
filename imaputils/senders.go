package imaputils

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"time"
)

// SenderCount holds information about a sender and the number of messages sent.
type SenderCount struct {
	Sender       string
	MessageCount int
}

// CountMessagesBySender counts messages by sender in the given folder,
// optionally restricted to a delivery-date range (startDate/endDate, either may
// be nil). Only senders with at least threshold messages are returned, sorted
// by descending count. Date filtering uses the same server-side INTERNALDATE
// search as the find command.
func CountMessagesBySender(dialer IMAPDialer, account Account, folder string, threshold int, startDate, endDate *time.Time) ([][]string, error) {
	criteria := BuildSearchCriteria(SearchOptions{StartDate: startDate, EndDate: endDate})
	messages, err := SearchMessages(dialer, account, folder, criteria)
	if err != nil {
		return nil, fmt.Errorf("error searching folder %s: %w", folder, err)
	}

	senderCounts := make(map[string]int, len(messages)/2)
	for _, message := range messages {
		if message.Envelope == nil || len(message.Envelope.From) == 0 {
			continue
		}
		sender := FormatAddress(message.Envelope.From[0])
		if sender != "" {
			senderCounts[sender]++
		}
	}

	senderCountList := make([]SenderCount, 0, len(senderCounts))
	for sender, count := range senderCounts {
		if count >= threshold {
			senderCountList = append(senderCountList, SenderCount{sender, count})
		}
	}

	// Sort by message count, descending.
	slices.SortFunc(senderCountList, func(a, b SenderCount) int {
		return cmp.Compare(b.MessageCount, a.MessageCount)
	})

	// Pre-allocate result slice
	tableData := make([][]string, 0, len(senderCountList)+1)
	tableData = append(tableData, []string{"Sender", "Number of Messages"})

	for _, senderCount := range senderCountList {
		tableData = append(tableData, []string{
			senderCount.Sender,
			strconv.Itoa(senderCount.MessageCount), // Use strconv.Itoa instead of fmt.Sprintf
		})
	}

	return tableData, nil
}

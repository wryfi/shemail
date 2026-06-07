package imaputils

import (
	"cmp"
	"fmt"
	"github.com/emersion/go-imap"
	"slices"
	"strconv"
	"sync"
)

// SenderCount holds information about a sender and the number of messages sent.
type SenderCount struct {
	Sender       string
	MessageCount int
}

// CountMessagesBySender counts the messages from a given sender with improved performance.
func CountMessagesBySender(dialer IMAPDialer, account Account, folder string, threshold int) ([][]string, error) {
	// Only fetch the fields we need for counting by sender
	fields := MessageFields{
		Envelope: true,
		Headers:  []string{"From"},
	}

	messages, err := FetchMessages(dialer, account, folder, fields)
	if err != nil {
		return nil, fmt.Errorf("error fetching messages from folder %s: %w", folder, err)
	}

	// Pre-allocate map with estimated capacity
	senderCounts := make(map[string]int, len(messages)/2) // Assuming average of 2 messages per sender

	// Process messages in parallel using worker pool
	const numWorkers = 4
	messageChan := make(chan *imap.Message, numWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex // Protects senderCounts

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for msg := range messageChan {
				if msg.Envelope == nil || len(msg.Envelope.From) == 0 {
					continue
				}

				sender := FormatAddress(msg.Envelope.From[0])

				if sender != "" {
					mu.Lock()
					senderCounts[sender]++
					mu.Unlock()
				}
			}
		}()
	}
	// Send messages to workers
	for _, msg := range messages {
		messageChan <- msg
	}
	close(messageChan)
	wg.Wait()

	// Pre-allocate slice with exact capacity
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

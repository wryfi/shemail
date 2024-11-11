package imap

import (
	"fmt"
	"github.com/emersion/go-imap"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// SenderCount holds information about a sender and the number of messages sent.
type SenderCount struct {
	Sender       string
	MessageCount int
}

// CountMessagesBySender counts the messages from a given sender with improved performance.
func CountMessagesBySender(account Account, folder string, threshold int) ([][]string, error) {
	// Only fetch the fields we need for counting by sender
	fields := MessageFields{
		Envelope: true,
		Headers:  []string{"From"},
		BodyPeek: true,
	}

	messages, err := FetchMessages(account, folder, fields)
	if err != nil {
		return nil, fmt.Errorf("error fetching messages from folder %s: %w", folder, err)
	}

	// Pre-allocate map with estimated capacity
	senderCounts := make(map[string]int, len(messages)/2) // Assuming average of 2 messages per sender

	// Use a sync.Pool for string builders to reduce allocations
	pool := sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}

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
				if len(msg.Envelope.From) == 0 {
					continue
				}

				sb := pool.Get().(*strings.Builder)
				sb.Reset()
				sender := FormatAddress(msg.Envelope.From[0]) // <-- Capture the return value
				pool.Put(sb)

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

	// Sort in parallel for large datasets
	if len(senderCountList) > 1000 {
		parallelSort(senderCountList)
	} else {
		sort.Slice(senderCountList, func(i, j int) bool {
			return senderCountList[i].MessageCount > senderCountList[j].MessageCount
		})
	}

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

// Helper function for parallel sorting of large datasets
func parallelSort(data []SenderCount) {
	if len(data) <= 1 {
		return
	}

	mid := len(data) / 2
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		parallelSort(data[:mid])
	}()
	parallelSort(data[mid:])
	wg.Wait()

	// Merge the sorted halves
	merged := make([]SenderCount, len(data))
	i, j, k := 0, mid, 0
	for i < mid && j < len(data) {
		if data[i].MessageCount > data[j].MessageCount {
			merged[k] = data[i]
			i++
		} else {
			merged[k] = data[j]
			j++
		}
		k++
	}
	copy(merged[k:], data[i:mid])
	copy(merged[k+mid-i:], data[j:])
	copy(data, merged)
}

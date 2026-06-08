package imaputils

import (
	"github.com/emersion/go-imap"
	"testing"
)

func messagesWithSubjects(subjects ...string) []*imap.Message {
	messages := make([]*imap.Message, 0, len(subjects))
	for _, subject := range subjects {
		messages = append(messages, &imap.Message{
			Envelope: &imap.Envelope{Subject: subject},
		})
	}
	return messages
}

func subjectsOf(messages []*imap.Message) []string {
	subjects := make([]string, 0, len(messages))
	for _, message := range messages {
		subjects = append(subjects, message.Envelope.Subject)
	}
	return subjects
}

func TestFilterBySubject(t *testing.T) {
	all := messagesWithSubjects(
		"Your tax forms are ready to download",
		"We're investing $20 in your Acorns account",
		"A lot's changed since we last saw you",
	)

	t.Run("no subject options returns all", func(t *testing.T) {
		result, err := FilterBySubject(all, SearchOptions{})
		if err != nil || len(result) != 3 {
			t.Fatalf("got %d messages, %v; want 3", len(result), err)
		}
	})

	t.Run("substring include is case-insensitive", func(t *testing.T) {
		result, err := FilterBySubject(all, SearchOptions{Subject: []string{"INVESTING"}})
		if err != nil {
			t.Fatal(err)
		}
		got := subjectsOf(result)
		if len(got) != 1 || got[0] != "We're investing $20 in your Acorns account" {
			t.Fatalf("got %v; want the investing message", got)
		}
	})

	t.Run("substring exclude drops matches", func(t *testing.T) {
		result, err := FilterBySubject(all, SearchOptions{NotSubject: []string{"tax forms"}})
		if err != nil {
			t.Fatal(err)
		}
		for _, message := range result {
			if message.Envelope.Subject == "Your tax forms are ready to download" {
				t.Fatalf("tax-forms message should have been excluded")
			}
		}
		if len(result) != 2 {
			t.Fatalf("got %d messages; want 2", len(result))
		}
	})

	t.Run("multiple includes match any", func(t *testing.T) {
		result, err := FilterBySubject(all, SearchOptions{
			Subject: []string{"tax forms", "investing"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 2 {
			t.Fatalf("got %d messages; want 2 (tax + investing)", len(result))
		}
	})

	t.Run("multiple excludes drop any", func(t *testing.T) {
		result, err := FilterBySubject(all, SearchOptions{
			NotSubject: []string{"tax forms", "investing"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 1 || result[0].Envelope.Subject != "A lot's changed since we last saw you" {
			t.Fatalf("got %v; want only the 'A lot's changed' message", subjectsOf(result))
		}
	})

	t.Run("regex include", func(t *testing.T) {
		result, err := FilterBySubject(all, SearchOptions{
			Subject:      []string{`\$\d+`},
			SubjectRegex: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 1 || result[0].Envelope.Subject != "We're investing $20 in your Acorns account" {
			t.Fatalf("got %v; want the $20 message", subjectsOf(result))
		}
	})

	t.Run("invalid regex errors", func(t *testing.T) {
		_, err := FilterBySubject(all, SearchOptions{
			Subject:      []string{"(unclosed"},
			SubjectRegex: true,
		})
		if err == nil {
			t.Fatal("expected an error for an invalid regex")
		}
	})

	t.Run("nil envelope is treated as empty subject", func(t *testing.T) {
		messages := []*imap.Message{{Envelope: nil}}
		// include should drop it (empty subject doesn't contain "x")
		included, _ := FilterBySubject(messages, SearchOptions{Subject: []string{"x"}})
		if len(included) != 0 {
			t.Fatalf("nil-envelope message should not match an include filter")
		}
		// exclude should keep it (empty subject doesn't contain "x")
		excluded, _ := FilterBySubject(messages, SearchOptions{NotSubject: []string{"x"}})
		if len(excluded) != 1 {
			t.Fatalf("nil-envelope message should survive an exclude filter")
		}
	})
}

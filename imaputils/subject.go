package imaputils

import (
	"fmt"
	"github.com/emersion/go-imap"
	"regexp"
	"strings"
)

// FilterBySubject filters messages by the subject options, matching against the
// decoded envelope subject.
//
// Subject matching is done client-side on purpose: server-side IMAP SUBJECT
// search is commonly backed by a full-text index (e.g. Dovecot fts), which is
// unreliable — especially for negation and for messages that aren't fully
// indexed yet. Matching the already-decoded Envelope.Subject here is
// deterministic and consistent with what shemail displays.
//
// Subject/NotSubject may each be given multiple patterns: a message is kept if
// its subject matches ANY of the Subject patterns (when any are given) and NONE
// of the NotSubject patterns. By default each pattern is a case-insensitive
// substring test; when opts.SubjectRegex is set they are treated as regular
// expressions (case-sensitive unless the pattern includes an inline flag such
// as "(?i)").
func FilterBySubject(messages []*imap.Message, opts SearchOptions) ([]*imap.Message, error) {
	if len(opts.Subject) == 0 && len(opts.NotSubject) == 0 {
		return messages, nil
	}

	includes, err := compileSubjectMatchers(opts.Subject, opts.SubjectRegex)
	if err != nil {
		return nil, fmt.Errorf("invalid subject pattern: %w", err)
	}
	excludes, err := compileSubjectMatchers(opts.NotSubject, opts.SubjectRegex)
	if err != nil {
		return nil, fmt.Errorf("invalid not-subject pattern: %w", err)
	}

	filtered := make([]*imap.Message, 0, len(messages))
	for _, message := range messages {
		subject := ""
		if message.Envelope != nil {
			subject = message.Envelope.Subject
		}

		if len(includes) > 0 && !matchesAny(includes, subject) {
			continue
		}
		if matchesAny(excludes, subject) {
			continue
		}
		filtered = append(filtered, message)
	}

	return filtered, nil
}

// matchesAny reports whether the subject satisfies at least one matcher.
func matchesAny(matchers []func(string) bool, subject string) bool {
	for _, matches := range matchers {
		if matches(subject) {
			return true
		}
	}
	return false
}

// compileSubjectMatchers compiles each pattern into a predicate. Substring
// matching is case-insensitive; regex matching uses the pattern as written.
func compileSubjectMatchers(patterns []string, useRegex bool) ([]func(string) bool, error) {
	matchers := make([]func(string) bool, 0, len(patterns))
	for _, pattern := range patterns {
		if useRegex {
			expression, err := regexp.Compile(pattern)
			if err != nil {
				return nil, err
			}
			matchers = append(matchers, expression.MatchString)
			continue
		}

		needle := strings.ToLower(pattern)
		matchers = append(matchers, func(subject string) bool {
			return strings.Contains(strings.ToLower(subject), needle)
		})
	}
	return matchers, nil
}

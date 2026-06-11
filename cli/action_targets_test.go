package cli

import (
	"testing"

	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
)

// These exercise the two non-interactive branches of resolveActionTargets. The
// test process's stdin is not a terminal, so isInteractive() is false here,
// which is exactly the "refuse without --yes" safety path we want to lock down.
func TestResolveActionTargets(t *testing.T) {
	messages := []*imap.Message{{Uid: 1}, {Uid: 2}}

	t.Run("--yes acts on all messages", func(t *testing.T) {
		targets, proceed, err := resolveActionTargets(messages, "delete", true)
		assert.NoError(t, err)
		assert.True(t, proceed)
		assert.Equal(t, messages, targets)
	})

	t.Run("non-interactive without --yes refuses", func(t *testing.T) {
		targets, proceed, err := resolveActionTargets(messages, "delete", false)
		assert.Error(t, err)
		assert.False(t, proceed)
		assert.Nil(t, targets)
		assert.Contains(t, err.Error(), "refusing to delete 2 messages")
		assert.Contains(t, err.Error(), "--yes")
	})
}

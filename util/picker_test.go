package util

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
)

func TestMessagePicker(t *testing.T) {
	messages := []*imap.Message{{Uid: 1}, {Uid: 2}, {Uid: 3}}
	rows := []MessageRow{
		{Cells: []string{"a", "b", "c", "d", "e"}, Unread: false},
		{Cells: []string{"a", "b", "c", "d", "e"}, Unread: true},
		{Cells: []string{"a", "b", "c", "d", "e"}, Unread: false},
	}

	runes := func(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }
	send := func(model tea.Model, msg tea.Msg) tea.Model {
		next, _ := model.Update(msg)
		return next
	}

	t.Run("all rows pre-selected", func(t *testing.T) {
		picker := newMessagePicker(messages, rows, "delete", false)
		assert.Equal(t, []bool{true, true, true}, picker.selected)
		assert.Equal(t, 3, picker.selectedCount())
		assert.Len(t, picker.keptMessages(), 3)
	})

	t.Run("space deselects the cursor row", func(t *testing.T) {
		var model tea.Model = newMessagePicker(messages, rows, "delete", false)
		model = send(model, tea.KeyMsg{Type: tea.KeySpace})
		picker := model.(messagePicker)
		assert.False(t, picker.selected[0])
		assert.Equal(t, 2, picker.selectedCount())
		assert.Equal(t, []*imap.Message{messages[1], messages[2]}, picker.keptMessages())
	})

	t.Run("navigate down then toggle", func(t *testing.T) {
		var model tea.Model = newMessagePicker(messages, rows, "delete", false)
		model = send(model, tea.KeyMsg{Type: tea.KeyDown})
		model = send(model, tea.KeyMsg{Type: tea.KeySpace})
		picker := model.(messagePicker)
		assert.Equal(t, 1, picker.cursor)
		assert.False(t, picker.selected[1])
	})

	t.Run("enter commits and quits", func(t *testing.T) {
		picker := newMessagePicker(messages, rows, "delete", false)
		next, cmd := picker.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assert.True(t, next.(messagePicker).committed)
		if assert.NotNil(t, cmd) {
			assert.IsType(t, tea.QuitMsg{}, cmd())
		}
	})

	t.Run("esc cancels without committing", func(t *testing.T) {
		picker := newMessagePicker(messages, rows, "delete", false)
		next, cmd := picker.Update(tea.KeyMsg{Type: tea.KeyEsc})
		assert.False(t, next.(messagePicker).committed)
		if assert.NotNil(t, cmd) {
			assert.IsType(t, tea.QuitMsg{}, cmd())
		}
	})

	t.Run("'a' toggles all off then back on", func(t *testing.T) {
		var model tea.Model = newMessagePicker(messages, rows, "delete", false)
		model = send(model, runes("a"))
		assert.Equal(t, 0, model.(messagePicker).selectedCount())
		model = send(model, runes("a"))
		assert.Equal(t, 3, model.(messagePicker).selectedCount())
	})

	t.Run("cursor clamps at both bounds", func(t *testing.T) {
		var model tea.Model = newMessagePicker(messages, rows, "delete", false)
		model = send(model, tea.KeyMsg{Type: tea.KeyUp}) // already at top
		assert.Equal(t, 0, model.(messagePicker).cursor)
		for index := 0; index < 10; index++ {
			model = send(model, tea.KeyMsg{Type: tea.KeyDown})
		}
		assert.Equal(t, 2, model.(messagePicker).cursor) // clamped at last row
	})

	t.Run("confirm-required action confirms before committing", func(t *testing.T) {
		picker := newMessagePicker(messages, rows, "delete", true)

		// First enter opens the confirm screen: not committed, no quit yet.
		next, cmd := picker.Update(tea.KeyMsg{Type: tea.KeyEnter})
		confirming := next.(messagePicker)
		assert.Equal(t, modeConfirming, confirming.mode)
		assert.False(t, confirming.committed)
		assert.Nil(t, cmd)

		// esc backs out to the list without committing.
		back, _ := confirming.Update(tea.KeyMsg{Type: tea.KeyEsc})
		assert.Equal(t, modeSelecting, back.(messagePicker).mode)
		assert.False(t, back.(messagePicker).committed)

		// A second enter from the confirm screen commits and quits.
		final, cmd := confirming.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assert.True(t, final.(messagePicker).committed)
		if assert.NotNil(t, cmd) {
			assert.IsType(t, tea.QuitMsg{}, cmd())
		}
	})

	t.Run("confirm-required action with empty selection skips the confirm", func(t *testing.T) {
		var model tea.Model = newMessagePicker(messages, rows, "delete", true)
		model = send(model, runes("a")) // deselect everything
		next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		// Nothing selected: commit straight away; the caller reports the no-op.
		assert.True(t, next.(messagePicker).committed)
		if assert.NotNil(t, cmd) {
			assert.IsType(t, tea.QuitMsg{}, cmd())
		}
	})
}

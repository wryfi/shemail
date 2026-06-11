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
		picker := newMessagePicker(messages, rows, "delete")
		assert.Equal(t, []bool{true, true, true}, picker.selected)
		assert.Equal(t, 3, picker.selectedCount())
		assert.Len(t, picker.keptMessages(), 3)
	})

	t.Run("space deselects the cursor row", func(t *testing.T) {
		var model tea.Model = newMessagePicker(messages, rows, "delete")
		model = send(model, tea.KeyMsg{Type: tea.KeySpace})
		picker := model.(messagePicker)
		assert.False(t, picker.selected[0])
		assert.Equal(t, 2, picker.selectedCount())
		assert.Equal(t, []*imap.Message{messages[1], messages[2]}, picker.keptMessages())
	})

	t.Run("navigate down then toggle", func(t *testing.T) {
		var model tea.Model = newMessagePicker(messages, rows, "delete")
		model = send(model, tea.KeyMsg{Type: tea.KeyDown})
		model = send(model, tea.KeyMsg{Type: tea.KeySpace})
		picker := model.(messagePicker)
		assert.Equal(t, 1, picker.cursor)
		assert.False(t, picker.selected[1])
	})

	t.Run("enter commits and quits", func(t *testing.T) {
		picker := newMessagePicker(messages, rows, "delete")
		next, cmd := picker.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assert.True(t, next.(messagePicker).committed)
		if assert.NotNil(t, cmd) {
			assert.IsType(t, tea.QuitMsg{}, cmd())
		}
	})

	t.Run("esc cancels without committing", func(t *testing.T) {
		picker := newMessagePicker(messages, rows, "delete")
		next, cmd := picker.Update(tea.KeyMsg{Type: tea.KeyEsc})
		assert.False(t, next.(messagePicker).committed)
		if assert.NotNil(t, cmd) {
			assert.IsType(t, tea.QuitMsg{}, cmd())
		}
	})

	t.Run("'a' toggles all off then back on", func(t *testing.T) {
		var model tea.Model = newMessagePicker(messages, rows, "delete")
		model = send(model, runes("a"))
		assert.Equal(t, 0, model.(messagePicker).selectedCount())
		model = send(model, runes("a"))
		assert.Equal(t, 3, model.(messagePicker).selectedCount())
	})

	t.Run("cursor clamps at both bounds", func(t *testing.T) {
		var model tea.Model = newMessagePicker(messages, rows, "delete")
		model = send(model, tea.KeyMsg{Type: tea.KeyUp}) // already at top
		assert.Equal(t, 0, model.(messagePicker).cursor)
		for index := 0; index < 10; index++ {
			model = send(model, tea.KeyMsg{Type: tea.KeyDown})
		}
		assert.Equal(t, 2, model.(messagePicker).cursor) // clamped at last row
	})
}

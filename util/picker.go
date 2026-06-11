package util

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	ltable "github.com/charmbracelet/lipgloss/table"
	"github.com/emersion/go-imap"
	"github.com/mattn/go-runewidth"
)

// checkboxColumnWidth is the width of the leading select column the picker
// prepends to the shared message columns.
const checkboxColumnWidth = 1

// Picker modes: the row list, or the final confirmation prompt.
const (
	modeSelecting = iota
	modeConfirming
)

var (
	pickerBase     = lipgloss.NewStyle().Padding(0, 1)
	pickerBold     = pickerBase.Bold(true)  // header and unread rows
	pickerMuted    = pickerBase.Faint(true) // read rows, de-emphasized
	pickerCursorBg = lipgloss.AdaptiveColor{Light: "252", Dark: "237"}
	pickerHelp     = lipgloss.NewStyle().Faint(true)
	pickerConfirm  = lipgloss.NewStyle().Bold(true)
)

// messagePicker is the Bubble Tea model backing SelectMessages. It renders the
// shared message table (via lipgloss/table, so styling matches the static view)
// plus a leading checkbox column, and lets the user toggle rows before
// committing. Scrolling is windowed manually so the header stays pinned.
type messagePicker struct {
	messages        []*imap.Message
	rows            []MessageRow
	selected        []bool
	action          string
	confirmRequired bool // state-changing actions get a final confirm screen
	mode            int
	cursor          int
	top             int // index of the first visible row
	height          int // number of message rows visible at once
	committed       bool
}

func newMessagePicker(messages []*imap.Message, rows []MessageRow, action string, confirmRequired bool) messagePicker {
	selected := make([]bool, len(messages))
	for index := range selected {
		selected[index] = true // everything pre-selected; the user deselects exceptions
	}
	return messagePicker{
		messages:        messages,
		rows:            rows,
		selected:        selected,
		action:          action,
		confirmRequired: confirmRequired,
		height:          20, // replaced by the first WindowSizeMsg
	}
}

func (picker messagePicker) Init() tea.Cmd { return nil }

func (picker messagePicker) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		// Reserve rows for the title, header+rule, and footer.
		picker.height = msg.Height - 6
		if picker.height < 1 {
			picker.height = 1
		}
		picker.clampScroll()
	case tea.KeyMsg:
		if picker.mode == modeConfirming {
			return picker.updateConfirming(msg)
		}
		return picker.updateSelecting(msg)
	}
	return picker, nil
}

func (picker messagePicker) updateSelecting(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc", "q":
		picker.committed = false
		return picker, tea.Quit
	case "enter":
		// State-changing actions get a final confirmation. An empty selection
		// just quits; the caller reports that nothing was selected.
		if picker.confirmRequired && picker.selectedCount() > 0 {
			picker.mode = modeConfirming
			return picker, nil
		}
		picker.committed = true
		return picker, tea.Quit
	case " ":
		if len(picker.selected) > 0 {
			picker.selected[picker.cursor] = !picker.selected[picker.cursor]
		}
	case "a":
		picker.toggleAll()
	case "up", "k":
		picker.move(-1)
	case "down", "j":
		picker.move(1)
	case "g", "home":
		picker.cursor = 0
		picker.clampScroll()
	case "G", "end":
		picker.cursor = len(picker.rows) - 1
		picker.clampScroll()
	}
	return picker, nil
}

func (picker messagePicker) updateConfirming(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		picker.committed = false
		return picker, tea.Quit
	case "enter", "y":
		picker.committed = true
		return picker, tea.Quit
	case "esc", "n":
		picker.mode = modeSelecting // back to the list to adjust the selection
	}
	return picker, nil
}

func (picker *messagePicker) move(delta int) {
	picker.cursor += delta
	if picker.cursor < 0 {
		picker.cursor = 0
	}
	if picker.cursor > len(picker.rows)-1 {
		picker.cursor = len(picker.rows) - 1
	}
	picker.clampScroll()
}

// clampScroll keeps the cursor within the visible window [top, top+height).
func (picker *messagePicker) clampScroll() {
	if picker.cursor < picker.top {
		picker.top = picker.cursor
	}
	if picker.cursor >= picker.top+picker.height {
		picker.top = picker.cursor - picker.height + 1
	}
	if picker.top < 0 {
		picker.top = 0
	}
}

func (picker *messagePicker) toggleAll() {
	// If everything is selected, clear; otherwise select all.
	allSelected := true
	for _, on := range picker.selected {
		if !on {
			allSelected = false
			break
		}
	}
	for index := range picker.selected {
		picker.selected[index] = !allSelected
	}
}

func (picker messagePicker) selectedCount() int {
	count := 0
	for _, on := range picker.selected {
		if on {
			count++
		}
	}
	return count
}

func (picker messagePicker) View() string {
	bottom := picker.top + picker.height
	if bottom > len(picker.rows) {
		bottom = len(picker.rows)
	}

	table := ltable.New().
		Border(lipgloss.NormalBorder()).
		BorderTop(false).BorderBottom(false).
		BorderLeft(false).BorderRight(false).
		BorderColumn(false).BorderRow(false).
		BorderHeader(true).
		Headers(append([]string{""}, MessageColumns...)...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == ltable.HeaderRow {
				return pickerBold
			}
			absolute := picker.top + row
			style := pickerMuted
			if absolute >= 0 && absolute < len(picker.rows) && picker.rows[absolute].Unread {
				style = pickerBold
			}
			if absolute == picker.cursor {
				// Background highlight composes cleanly with bold; drop faint so
				// the highlighted row stays legible.
				style = pickerBase.Background(pickerCursorBg)
				if absolute < len(picker.rows) && picker.rows[absolute].Unread {
					style = style.Bold(true)
				}
			}
			return style
		})

	for index := picker.top; index < bottom; index++ {
		check := " "
		if picker.selected[index] {
			check = "✓"
		}
		// Pad cells to fixed widths so columns stay put as rows scroll (the
		// resizer auto-sizes to uniform content; .Width() would fight padding).
		cells := append([]string{padCell(check, checkboxColumnWidth)}, padCells(picker.rows[index].Cells)...)
		table.Row(cells...)
	}

	if picker.mode == modeConfirming {
		title := fmt.Sprintf("Confirm — %s", picker.action)
		prompt := pickerConfirm.Render(fmt.Sprintf("really %s %d messages?  enter: yes · esc: back",
			picker.action, picker.selectedCount()))
		return strings.Join([]string{title, table.String(), prompt}, "\n")
	}

	title := fmt.Sprintf("Select messages to %s — %d of %d selected",
		picker.action, picker.selectedCount(), len(picker.rows))
	help := pickerHelp.Render("↑/↓ move · space toggle · a all/none · enter confirm · esc cancel")
	return strings.Join([]string{title, table.String(), help}, "\n")
}

// padCell truncates (defensively) and right-pads a cell to an exact display
// width so every cell in a column is the same width.
func padCell(value string, width int) string {
	return runewidth.FillRight(runewidth.Truncate(value, width, ""), width)
}

// padCells pads the message cells to their shared column widths.
func padCells(cells []string) []string {
	padded := make([]string, len(cells))
	for index, cell := range cells {
		width := 0
		if index < len(messageColumnWidths) {
			width = messageColumnWidths[index]
		}
		padded[index] = padCell(cell, width)
	}
	return padded
}

// SelectMessages runs the interactive picker over messages (all pre-selected)
// and returns the subset the user keeps. committed is false when the user
// cancels, in which case the caller must not act. action labels the pending
// operation in the picker header (e.g. "delete", "move to Archive").
// confirmRequired adds a final confirmation screen for state-changing actions
// (copy/move/delete); pass false for trivially reversible ones (mark read).
func SelectMessages(messages []*imap.Message, action string, confirmRequired bool) (kept []*imap.Message, committed bool, err error) {
	if len(messages) == 0 {
		return nil, false, nil
	}
	rows, err := FormatMessageRows(messages)
	if err != nil {
		return nil, false, err
	}

	final, err := tea.NewProgram(newMessagePicker(messages, rows, action, confirmRequired), tea.WithAltScreen()).Run()
	if err != nil {
		return nil, false, fmt.Errorf("interactive selection failed: %w", err)
	}

	picker := final.(messagePicker)
	if !picker.committed {
		return nil, false, nil
	}
	return picker.keptMessages(), true, nil
}

// keptMessages returns the messages the user left selected.
func (picker messagePicker) keptMessages() []*imap.Message {
	var kept []*imap.Message
	for index, message := range picker.messages {
		if picker.selected[index] {
			kept = append(kept, message)
		}
	}
	return kept
}

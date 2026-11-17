package arazzoexplore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;:]*[A-Za-z]`)

// testItems is two workflows with two steps each.
func testItems() []Item {
	return []Item{
		{Kind: KindWorkflow, Label: "adoptPet", Summary: "Adopt a pet", Details: []string{"Summary: Adopt a pet", "Steps: 2"}},
		{Kind: KindStep, Label: "findPet", Badge: BadgeOperation, Summary: "listPets", Details: []string{"Operation: listPets"}},
		{Kind: KindStep, Label: "adopt", Badge: BadgeOperationPath, Summary: "#/paths/~1pets/post", Details: []string{"Operation path: #/paths/~1pets/post"}},
		{Kind: KindWorkflow, Label: "registerOwner", Details: []string{"Steps: 2"}},
		{Kind: KindStep, Label: "create", Badge: BadgeOperation, Summary: "createOwner"},
		{Kind: KindStep, Label: "notify", Badge: BadgeSend, Summary: "#/channels/owners"},
	}
}

// manyItems is a workflow with enough steps to scroll, each with three detail lines.
func manyItems() []Item {
	items := []Item{{Kind: KindWorkflow, Label: "big", Details: []string{"Steps: 30"}}}
	for i := range 30 {
		items = append(items, Item{Kind: KindStep, Label: "step" + strconv.Itoa(i), Badge: BadgeOperation, Details: []string{"a", "b", "c"}})
	}
	return items
}

func newTestModel(items []Item) Model {
	return NewModel(items, "Pets", "1.0.0", "1.1.0")
}

func update(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)
	next, ok := updated.(Model)
	require.True(t, ok, "Update should return a Model")
	return next, cmd
}

// press sends key presses in order.
func press(t *testing.T, m Model, keys ...tea.KeyPressMsg) Model {
	t.Helper()

	for _, key := range keys {
		m, _ = update(t, m, key)
	}
	return m
}

func key(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

func ctrl(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Mod: tea.ModCtrl}
}

func enterKey() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyEnter}
}

func spaceKey() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
}

func resize(t *testing.T, m Model, width, height int) Model {
	t.Helper()

	m, _ = update(t, m, tea.WindowSizeMsg{Width: width, Height: height})
	return m
}

// render returns the view without ANSI styling.
func render(m Model) string {
	return ansiRegex.ReplaceAllString(m.View().Content, "")
}

// assertCursorRendered checks the cursor row is drawn in the content area.
func assertCursorRendered(t *testing.T, m Model) {
	t.Helper()

	rows := ansiRegex.ReplaceAllString(strings.Join(m.renderRows(), "\n"), "")
	assert.Contains(t, rows, m.items[m.cursor].Label, "the cursor row should be visible")
	assert.Len(t, m.renderRows(), m.contentHeight(), "rows should fill the content area exactly")
}

func TestModel_Navigation_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		keys     []tea.KeyPressMsg
		expected int
	}{
		{name: "down", keys: []tea.KeyPressMsg{key('j'), {Code: tea.KeyDown}}, expected: 2},
		{name: "up stops at the top", keys: []tea.KeyPressMsg{key('j'), key('k'), {Code: tea.KeyUp}}, expected: 0},
		{name: "down stops at the bottom", keys: []tea.KeyPressMsg{key('G'), key('j')}, expected: 5},
		{name: "gg jumps to the top", keys: []tea.KeyPressMsg{key('G'), key('g'), key('g')}, expected: 0},
		{name: "a single g doesn't move", keys: []tea.KeyPressMsg{key('G'), key('g')}, expected: 5},
		{name: "another key interrupts gg", keys: []tea.KeyPressMsg{key('G'), key('g'), key('e'), key('g')}, expected: 5},
		{name: "a key swallowed by help interrupts gg", keys: []tea.KeyPressMsg{key('G'), key('g'), key('?'), key('?'), key('g')}, expected: 5},
		{name: "next workflow", keys: []tea.KeyPressMsg{key(']')}, expected: 3},
		{name: "next workflow stays on the last", keys: []tea.KeyPressMsg{key(']'), key(']')}, expected: 3},
		{name: "previous workflow from a step", keys: []tea.KeyPressMsg{key('G'), key('[')}, expected: 3},
		{name: "previous workflow from a workflow", keys: []tea.KeyPressMsg{key(']'), key('[')}, expected: 0},
		{name: "previous workflow stays on the first", keys: []tea.KeyPressMsg{key('[')}, expected: 0},
		{name: "half page down", keys: []tea.KeyPressMsg{ctrl('d')}, expected: 5},
		{name: "half page up", keys: []tea.KeyPressMsg{key('G'), ctrl('u')}, expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := press(t, newTestModel(testItems()), tt.keys...)
			assert.Equal(t, tt.expected, m.cursor, "cursor should move to the expected row")
		})
	}
}

func TestModel_ToggleDetails_Success(t *testing.T) {
	t.Parallel()

	m := newTestModel(testItems())

	m = press(t, m, enterKey())
	assert.True(t, m.expanded[0], "enter should expand the cursor row")
	assert.Contains(t, render(m), "▼ adoptPet", "an expanded row should show the expanded icon")
	assert.Contains(t, render(m), "Summary: Adopt a pet", "an expanded row should show its details")

	m = press(t, m, spaceKey())
	assert.False(t, m.expanded[0], "space should collapse it again")
	assert.NotContains(t, render(m), "Summary: Adopt a pet", "a collapsed row should hide its details")

	m = press(t, m, key('e'))
	assert.Len(t, m.expanded, len(testItems()), "e should expand every row")
	assert.Contains(t, render(m), "Operation path: #/paths/~1pets/post", "step details should be shown")

	m = press(t, m, key('E'))
	assert.Empty(t, m.expanded, "E should collapse every row")
}

func TestModel_Help_Success(t *testing.T) {
	t.Parallel()

	m := press(t, newTestModel(testItems()), key('?'))
	assert.True(t, m.showHelp, "? should open the help")
	assert.Contains(t, render(m), "Previous / next workflow", "the help should list the key bindings")

	m = press(t, m, key('j'), enterKey())
	assert.Equal(t, 0, m.cursor, "the help should swallow navigation keys")
	assert.Empty(t, m.expanded, "the help should swallow toggle keys")

	tests := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{name: "question mark", key: key('?')},
		{name: "escape", key: tea.KeyPressMsg{Code: tea.KeyEscape}},
		{name: "q", key: key('q')},
		{name: "ctrl+c", key: ctrl('c')},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			closed, cmd := update(t, m, tt.key)
			assert.False(t, closed.showHelp, "the key should close the help")
			assert.Nil(t, cmd, "closing the help shouldn't quit")
		})
	}
}

func TestModel_Quit_Success(t *testing.T) {
	t.Parallel()

	for _, k := range []tea.KeyPressMsg{key('q'), ctrl('c')} {
		_, cmd := update(t, newTestModel(testItems()), k)
		require.NotNil(t, cmd, "the key should quit")
		assert.Equal(t, tea.Quit(), cmd(), "the command should quit")
	}

	m, cmd := update(t, newTestModel(testItems()), tea.KeyPressMsg{Code: tea.KeyEscape})
	assert.Nil(t, cmd, "escape outside the help shouldn't quit")
	assert.False(t, m.showHelp, "escape should leave the help closed")
}

func TestModel_Scrolling_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		height int
		keys   []tea.KeyPressMsg
	}{
		{name: "to the bottom", height: 12, keys: []tea.KeyPressMsg{key('G')}},
		{name: "expanded rows", height: 12, keys: []tea.KeyPressMsg{key('e'), key('j'), key('j'), key('j'), key('j')}},
		{name: "back to the top", height: 12, keys: []tea.KeyPressMsg{key('e'), key('G'), key('g'), key('g')}},
		{name: "half pages", height: 15, keys: []tea.KeyPressMsg{ctrl('d'), ctrl('d'), ctrl('u')}},
		{name: "expanding the last row", height: 12, keys: []tea.KeyPressMsg{key('G'), enterKey()}},
		{name: "a single line of content", height: 5, keys: []tea.KeyPressMsg{key('e'), key('j'), key('j'), key('G')}},
		{name: "two lines of content", height: 6, keys: []tea.KeyPressMsg{key('j'), key('j'), key('j')}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := resize(t, newTestModel(manyItems()), 80, tt.height)
			for _, k := range tt.keys {
				m = press(t, m, k)
				assertCursorRendered(t, m)
			}
		})
	}
}

func TestModel_ScrollIndicators_Success(t *testing.T) {
	t.Parallel()

	m := resize(t, newTestModel(manyItems()), 80, 12)
	assert.NotContains(t, render(m), "More above", "the top shouldn't show the above indicator")
	assert.Contains(t, render(m), "More below", "a long list should show the below indicator")

	m = press(t, m, key('G'))
	assert.Contains(t, render(m), "More above", "the bottom should show the above indicator")
	assert.NotContains(t, render(m), "More below", "the bottom shouldn't show the below indicator")

	m = press(t, m, enterKey())
	assert.Contains(t, render(m), "More below", "details cut off at the bottom should show the below indicator")
}

func TestModel_View_Success(t *testing.T) {
	t.Parallel()

	m := resize(t, newTestModel(testItems()), 100, 20)
	view := render(m)

	assert.Contains(t, view, "Workflows", "the header should show the mode")
	assert.Contains(t, view, "Arazzo Explorer", "the header should show the title")
	assert.Contains(t, view, "▶ adoptPet  Adopt a pet", "workflow rows should show the summary")
	assert.Contains(t, view, "  ▶ OP   findPet  listPets", "step rows should show the badge and target")
	assert.Contains(t, view, "  ▶ SEND notify  #/channels/owners", "send steps should have a badge")
	assert.Contains(t, view, "Press '?' for help", "the footer should show the help hint")
	assert.Contains(t, view, "Pets v1.0.0 · Arazzo 1.1.0 · 2 workflows", "the footer should show the document")
	assert.Equal(t, 20, strings.Count(view, "\n")+1, "the view should fill the terminal")

	narrow := render(resize(t, m, 30, 20))
	assert.NotContains(t, narrow, "Arazzo Explorer", "a narrow header should drop the title")
	assert.NotContains(t, narrow, "Press '?' for help", "a narrow footer should drop the help hint")
	for _, line := range strings.Split(narrow, "\n") {
		assert.LessOrEqual(t, len([]rune(line)), 30, "lines should be truncated to the terminal width")
	}
}

func TestModel_EmptyItems_Success(t *testing.T) {
	t.Parallel()

	m := press(t, newTestModel(nil), key('j'), key('G'), enterKey(), key(']'))
	assert.Equal(t, 0, m.cursor, "the cursor should stay put without items")
	assert.Contains(t, render(m), "0 workflows", "an empty explorer should still render")
	assert.Nil(t, m.Init(), "init shouldn't run a command")
}

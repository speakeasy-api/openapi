package tui

import (
	"maps"
	"slices"
	"strconv"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/speakeasy-api/openapi/cmd/openapi/internal/explore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testOperations() []explore.OperationInfo {
	return []explore.OperationInfo{
		{Path: "/users", Method: "GET", OperationID: "getUsers", Folded: true},
		{Path: "/users", Method: "POST", OperationID: "createUser", Folded: true},
		{Path: "/posts", Method: "GET", OperationID: "getPosts", Folded: true},
	}
}

func press(t *testing.T, m Model, key tea.KeyPressMsg) (Model, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(key)
	next, ok := updated.(Model)
	require.True(t, ok, "Update should return a Model")

	return next, cmd
}

func spaceKey() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
}

// assertCursorOnScreen checks both viewport bounds: the cursor row is not above
// the scroll offset and not below the last visible row for the current height.
func assertCursorOnScreen(t *testing.T, m Model) {
	t.Helper()
	assert.GreaterOrEqual(t, m.scrollOffset, 0)
	assert.LessOrEqual(t, m.scrollOffset, m.cursor, "cursor scrolled above the viewport")
	assert.LessOrEqual(t, m.cursor, m.scrollOffset+m.calculateContentHeight()-1, "cursor scrolled below the viewport")
}

func charKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

func ctrlKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Mod: tea.ModCtrl}
}

func enterKey() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyEnter}
}

// manyOperationCount is enough rows to scroll and half-page through.
const manyOperationCount = 30

func manyOperations() []explore.OperationInfo {
	ops := make([]explore.OperationInfo, manyOperationCount)
	for i := range ops {
		ops[i] = explore.OperationInfo{
			Path:        "/items/" + strconv.Itoa(i),
			Method:      "GET",
			OperationID: "getItem" + strconv.Itoa(i),
			Folded:      true,
		}
	}
	return ops
}

func selectionConfig(actionKeys ...ActionKey) Config {
	config := DefaultConfig()
	config.Selection.Enabled = true
	config.Selection.ActionKeys = actionKeys
	return config
}

// resize delivers a WindowSizeMsg so height-dependent handlers see a known terminal.
func resize(t *testing.T, m Model, height int) Model {
	t.Helper()

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: height})
	next, ok := updated.(Model)
	require.True(t, ok, "Update should return a Model")
	require.Equal(t, height, next.height)

	return next
}

// snapshot copies the slice and map that Update shares with the original so
// assert.Equal against it sees in-place mutations of fold or selection state.
func snapshot(m Model) Model {
	m.operations = slices.Clone(m.operations)
	m.selected = maps.Clone(m.selected)
	return m
}

func TestUpdate_SpaceTogglesFoldInViewMode(t *testing.T) {
	t.Parallel()

	m := NewModel(testOperations(), "Test API", "1.0.0")
	require.True(t, m.operations[0].Folded)

	m, _ = press(t, m, spaceKey())
	assert.False(t, m.operations[0].Folded, "space should unfold the cursor row")

	m, _ = press(t, m, spaceKey())
	assert.True(t, m.operations[0].Folded, "space should fold it again")
}

func TestUpdate_SpaceTogglesSelectionInSelectionMode(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	config.Selection.Enabled = true
	m := NewModelWithConfig(testOperations(), "Test API", "1.0.0", config)

	m, _ = press(t, m, spaceKey())
	selected := m.GetSelectedOperations()
	require.Len(t, selected, 1)
	assert.Equal(t, "getUsers", selected[0].OperationID)
	assert.True(t, m.operations[0].Folded, "space must not touch fold state in selection mode")

	m, _ = press(t, m, spaceKey())
	assert.Empty(t, m.GetSelectedOperations(), "second space should deselect")
}

func TestUpdate_NavigationKeysMoveCursor(t *testing.T) {
	t.Parallel()

	m := NewModel(testOperations(), "Test API", "1.0.0")

	m, _ = press(t, m, tea.KeyPressMsg{Code: 'j', Text: "j"})
	assert.Equal(t, 1, m.cursor)

	m, _ = press(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	assert.Equal(t, 2, m.cursor)

	m, _ = press(t, m, tea.KeyPressMsg{Code: 'j', Text: "j"})
	assert.Equal(t, 2, m.cursor, "cursor should stop at the last row")

	m, _ = press(t, m, tea.KeyPressMsg{Code: 'k', Text: "k"})
	assert.Equal(t, 1, m.cursor)

	m, _ = press(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	assert.Equal(t, 0, m.cursor)

	m, _ = press(t, m, tea.KeyPressMsg{Code: 'k', Text: "k"})
	assert.Equal(t, 0, m.cursor, "cursor should stop at the first row")
}

func TestUpdate_ActionKeyQuitsWithKey(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	config.Selection.Enabled = true
	config.Selection.ActionKeys = []ActionKey{{Key: "w", Label: "Write and save"}}
	m := NewModelWithConfig(testOperations(), "Test API", "1.0.0", config)

	m, cmd := press(t, m, tea.KeyPressMsg{Code: 'w', Text: "w"})
	assert.Equal(t, "w", m.GetActionKey())
	require.NotNil(t, cmd)
	assert.IsType(t, tea.QuitMsg{}, cmd())
}

func TestUpdate_QuitKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{name: "q", key: tea.KeyPressMsg{Code: 'q', Text: "q"}},
		{name: "ctrl+c", key: tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewModel(testOperations(), "Test API", "1.0.0")

			m, cmd := press(t, m, tt.key)
			assert.True(t, m.quitting)
			require.NotNil(t, cmd)
			assert.IsType(t, tea.QuitMsg{}, cmd())
			assert.Empty(t, m.GetActionKey())
		})
	}
}

func TestUpdate_QuitKeysCloseHelpFirst(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{name: "q", key: charKey('q')},
		{name: "ctrl+c", key: ctrlKey('c')},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewModel(testOperations(), "Test API", "1.0.0")
			m.showHelp = true

			m, cmd := press(t, m, tt.key)
			assert.False(t, m.showHelp, "first press should close help")
			assert.False(t, m.quitting, "closing help must not quit")
			assert.Nil(t, cmd)
		})
	}
}

func TestUpdate_QuestionMarkTogglesHelp(t *testing.T) {
	t.Parallel()

	m := NewModel(testOperations(), "Test API", "1.0.0")
	require.False(t, m.showHelp)

	m, _ = press(t, m, charKey('?'))
	assert.True(t, m.showHelp, "? should open help")

	m, _ = press(t, m, charKey('?'))
	assert.False(t, m.showHelp, "? should close help again")
}

func TestUpdate_EscOnlyClosesHelp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		showHelp bool
	}{
		{name: "closes help when open", showHelp: true},
		{name: "no-op when help is closed", showHelp: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewModel(testOperations(), "Test API", "1.0.0")
			m.cursor = 1
			m.showHelp = tt.showHelp
			want := snapshot(m)
			want.showHelp = false

			m, cmd := press(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
			assert.Nil(t, cmd)
			assert.Equal(t, want, m, "esc must touch nothing but the help flag")
		})
	}
}

func TestUpdate_EnterTogglesDetailsInBothModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config Config
	}{
		{name: "view mode", config: DefaultConfig()},
		{name: "selection mode", config: selectionConfig()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewModelWithConfig(testOperations(), "Test API", "1.0.0", tt.config)
			m.cursor = 1

			m, _ = press(t, m, enterKey())
			assert.False(t, m.operations[1].Folded, "enter should unfold the cursor row")
			assert.True(t, m.operations[0].Folded, "other rows stay folded")
			assert.Empty(t, m.GetSelectedOperations(), "enter must never select")

			m, _ = press(t, m, enterKey())
			assert.True(t, m.operations[1].Folded, "enter should fold it again")
		})
	}
}

func TestUpdate_JumpKeys(t *testing.T) {
	t.Parallel()

	const lastRow = manyOperationCount - 1

	tests := []struct {
		name         string
		ops          []explore.OperationInfo
		cursor       int
		showHelp     bool
		lastKey      string
		lastKeyAt    time.Time
		key          tea.KeyPressMsg
		wantCursor   int
		wantScrolled bool
		wantArmed    bool
	}{
		{
			name:         "G jumps to the last operation",
			ops:          manyOperations(),
			key:          charKey('G'),
			wantCursor:   lastRow,
			wantScrolled: true,
		},
		{
			name:       "G on an empty list is a no-op",
			key:        charKey('G'),
			wantCursor: 0,
		},
		{
			name:       "single g arms the sequence without moving",
			ops:        manyOperations(),
			cursor:     12,
			key:        charKey('g'),
			wantCursor: 12,
			wantArmed:  true,
		},
		{
			name:       "gg within the window jumps to the top",
			ops:        manyOperations(),
			cursor:     lastRow,
			lastKey:    "g",
			lastKeyAt:  time.Now(),
			key:        charKey('g'),
			wantCursor: 0,
		},
		{
			name:         "g after the window expired re-arms instead of jumping",
			ops:          manyOperations(),
			cursor:       lastRow,
			lastKey:      "g",
			lastKeyAt:    time.Now().Add(-keySequenceThreshold),
			key:          charKey('g'),
			wantCursor:   lastRow,
			wantScrolled: true,
			wantArmed:    true,
		},
		{
			name:         "gg with help open consumes the sequence without moving",
			ops:          manyOperations(),
			cursor:       lastRow,
			showHelp:     true,
			lastKey:      "g",
			lastKeyAt:    time.Now(),
			key:          charKey('g'),
			wantCursor:   lastRow,
			wantScrolled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewModel(tt.ops, "Test API", "1.0.0")
			m.cursor = tt.cursor
			m.ensureCursorVisible()
			m.showHelp = tt.showHelp
			m.lastKey = tt.lastKey
			m.lastKeyAt = tt.lastKeyAt

			m, cmd := press(t, m, tt.key)
			assert.Nil(t, cmd)
			assert.Equal(t, tt.wantCursor, m.cursor)
			assert.Equal(t, tt.showHelp, m.showHelp)

			if tt.wantScrolled {
				assert.Positive(t, m.scrollOffset, "view should scroll to keep the cursor visible")
			} else {
				assert.Equal(t, 0, m.scrollOffset)
			}
			assertCursorOnScreen(t, m)

			if tt.wantArmed {
				assert.Equal(t, "g", m.lastKey)
				assert.False(t, m.lastKeyAt.IsZero())
			} else {
				assert.Empty(t, m.lastKey)
				assert.True(t, m.lastKeyAt.IsZero())
			}
		})
	}
}

func TestUpdate_HalfPageKeys(t *testing.T) {
	t.Parallel()

	// A 28-row terminal leaves 20 content rows, so ctrl+u moves 10.
	const (
		height      = 28
		halfContent = 10
		lastRow     = manyOperationCount - 1
	)

	tests := []struct {
		name       string
		height     int
		cursor     int
		key        tea.KeyPressMsg
		wantCursor int
	}{
		{name: "ctrl+d moves down a half screen", height: height, cursor: 0, key: ctrlKey('d'), wantCursor: scrollHalfScreenLines},
		{name: "ctrl+d clamps at the last operation", height: height, cursor: 15, key: ctrlKey('d'), wantCursor: lastRow},
		{name: "ctrl+d at the last operation stays put", height: height, cursor: lastRow, key: ctrlKey('d'), wantCursor: lastRow},
		{name: "ctrl+u moves up half the content height", height: height, cursor: 25, key: ctrlKey('u'), wantCursor: 25 - halfContent},
		{name: "ctrl+u clamps at the first operation", height: height, cursor: 5, key: ctrlKey('u'), wantCursor: 0},
		{name: "ctrl+u at the first operation stays put", height: height, cursor: 0, key: ctrlKey('u'), wantCursor: 0},
		// At height 8 the content area is a single row.
		{name: "ctrl+u still moves one row on a tiny terminal", height: 8, cursor: 5, key: ctrlKey('u'), wantCursor: 4},
		{name: "ctrl+d still moves down on a tiny terminal", height: 8, cursor: 0, key: ctrlKey('d'), wantCursor: scrollHalfScreenLines},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := resize(t, NewModel(manyOperations(), "Test API", "1.0.0"), tt.height)
			m.cursor = tt.cursor
			m.ensureCursorVisible()

			m, cmd := press(t, m, tt.key)
			assert.Nil(t, cmd)
			assert.Equal(t, tt.wantCursor, m.cursor)
			assertCursorOnScreen(t, m)
		})
	}
}

func TestUpdate_HalfPageDownOnEmptyList(t *testing.T) {
	t.Parallel()

	m := resize(t, NewModel(nil, "Test API", "1.0.0"), 28)

	m, cmd := press(t, m, ctrlKey('d'))
	assert.Nil(t, cmd)
	assert.Equal(t, 0, m.cursor)
	assert.Equal(t, 0, m.scrollOffset)
}

func TestEnsureCursorVisible_SingleRowContentArea(t *testing.T) {
	t.Parallel()

	// height 8 leaves one content row, so the "more items above" indicator and
	// the cursor row cannot both fit; the cursor row must win.
	m := resize(t, NewModel(manyOperations(), "Test API", "1.0.0"), 8)
	assert.Equal(t, 1, m.calculateContentHeight())

	for _, cursor := range []int{1, 5, manyOperationCount - 1} {
		m.cursor = cursor
		m.ensureCursorVisible()
		assert.Equal(t, cursor, m.scrollOffset, "cursor %d", cursor)
		assertCursorOnScreen(t, m)
	}
}

func TestEnsureCursorVisible_UnfoldedItemTallerThanContentArea(t *testing.T) {
	t.Parallel()

	// height 10 leaves two content rows; an unfolded item spans more than that,
	// so the cursor row must be placed at the top rather than left off-screen.
	m := resize(t, NewModel(manyOperations(), "Test API", "1.0.0"), 10)
	assert.Equal(t, 2, m.calculateContentHeight())

	m.cursor = 5
	m.operations[5].Folded = false
	assert.Greater(t, m.getItemHeight(5), m.calculateContentHeight())

	m.ensureCursorVisible()
	assert.Equal(t, 5, m.scrollOffset)
	assertCursorOnScreen(t, m)
}

func TestUpdate_SelectAllKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		config       Config
		preselected  []int
		key          tea.KeyPressMsg
		wantSelected int
	}{
		{name: "a selects every operation in selection mode", config: selectionConfig(), key: charKey('a'), wantSelected: 3},
		{name: "A clears the selection in selection mode", config: selectionConfig(), preselected: []int{0, 1, 2}, key: charKey('A'), wantSelected: 0},
		{name: "a is a no-op in view mode", config: DefaultConfig(), key: charKey('a'), wantSelected: 0},
		{name: "A is a no-op in view mode", config: DefaultConfig(), key: charKey('A'), wantSelected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewModelWithConfig(testOperations(), "Test API", "1.0.0", tt.config)
			for _, i := range tt.preselected {
				m.selected[i] = true
			}
			before := snapshot(m)

			m, cmd := press(t, m, tt.key)
			assert.Nil(t, cmd)
			assert.Len(t, m.GetSelectedOperations(), tt.wantSelected)
			for i, op := range m.operations {
				assert.Truef(t, op.Folded, "row %d must stay folded", i)
			}
			if !tt.config.Selection.Enabled {
				assert.Equal(t, before, m, "select-all keys must not touch view mode")
			}
		})
	}
}

func TestUpdate_ActionKeyIgnoredOutsideSelectionMode(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	config.Selection.ActionKeys = []ActionKey{{Key: "w", Label: "Write and save"}}
	m := NewModelWithConfig(testOperations(), "Test API", "1.0.0", config)
	before := snapshot(m)

	m, cmd := press(t, m, charKey('w'))
	assert.Nil(t, cmd)
	assert.Equal(t, before, m, "action keys only apply in selection mode")
}

func TestUpdate_HelpModalSwallowsKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{name: "j", key: charKey('j')},
		{name: "k", key: charKey('k')},
		{name: "down", key: tea.KeyPressMsg{Code: tea.KeyDown}},
		{name: "up", key: tea.KeyPressMsg{Code: tea.KeyUp}},
		{name: "ctrl+d", key: ctrlKey('d')},
		{name: "ctrl+u", key: ctrlKey('u')},
		{name: "G", key: charKey('G')},
		{name: "space", key: spaceKey()},
		{name: "enter", key: enterKey()},
		{name: "a", key: charKey('a')},
		{name: "A", key: charKey('A')},
		{name: "action key w", key: charKey('w')},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := selectionConfig(ActionKey{Key: "w", Label: "Write and save"})
			m := NewModelWithConfig(testOperations(), "Test API", "1.0.0", config)
			m.cursor = 1
			m.showHelp = true
			before := snapshot(m)

			m, cmd := press(t, m, tt.key)
			assert.Nil(t, cmd)
			assert.Equal(t, before, m, "key must be ignored while help is open")
		})
	}
}

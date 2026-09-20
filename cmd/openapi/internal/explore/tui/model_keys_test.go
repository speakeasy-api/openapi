package tui

import (
	"testing"

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

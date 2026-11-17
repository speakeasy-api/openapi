package arazzo

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRules_Text_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flags    listRulesFlags
		expected []string
	}{
		{
			name:     "all rules",
			flags:    listRulesFlags{},
			expected: []string{"SEMANTIC (5 rules)", "STYLE (5 rules)", "SECURITY (1 rules)", "[fixable]", "11 rules total"},
		},
		{
			name:     "by category",
			flags:    listRulesFlags{category: "security"},
			expected: []string{"security-source-description-https", "1 rules total"},
		},
		{
			name:     "by ruleset",
			flags:    listRulesFlags{ruleset: "recommended"},
			expected: []string{"8 rules total"},
		},
		{
			name:     "no matches",
			flags:    listRulesFlags{category: "unknown"},
			expected: []string{"No rules found matching the specified filters."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			require.NoError(t, listRules(&buf, tt.flags), "list rules should succeed")
			for _, expected := range tt.expected {
				assert.Contains(t, buf.String(), expected, "output should contain expected text")
			}
		})
	}
}

func TestListRules_JSON_Success(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, listRules(&buf, listRulesFlags{format: "json"}), "list rules should succeed")

	var infos []ruleInfo
	require.NoError(t, json.Unmarshal(buf.Bytes(), &infos), "output should be JSON")
	assert.Len(t, infos, 11, "all rules should be listed")
	assert.NotNil(t, GetListRulesCommand(), "list-rules command should be exposed")
}

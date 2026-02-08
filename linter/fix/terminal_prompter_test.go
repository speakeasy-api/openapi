package fix_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi/linter/fix"
	"github.com/speakeasy-api/openapi/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestTerminalPrompter_Choice_Success(t *testing.T) {
	t.Parallel()

	input := strings.NewReader("2\n")
	output := &bytes.Buffer{}
	prompter := fix.NewTerminalPrompter(input, output)

	finding := &validation.Error{
		UnderlyingError: errors.New("missing tag"),
		Node:            &yaml.Node{Line: 10, Column: 5},
		Rule:            "test-rule",
	}
	f := &mockInteractiveFix{
		description: "choose a tag",
		prompts: []validation.Prompt{
			{
				Type:    validation.PromptChoice,
				Message: "Select a tag:",
				Choices: []string{"users", "accounts", "admin"},
			},
		},
	}

	responses, err := prompter.PromptFix(finding, f)
	require.NoError(t, err, "PromptFix should not fail")
	assert.Equal(t, []string{"accounts"}, responses, "should return the selected choice")
	assert.Contains(t, output.String(), "choose a tag", "should display fix description")
	assert.Contains(t, output.String(), "[1] users", "should display choices")
}

func TestTerminalPrompter_Choice_Default(t *testing.T) {
	t.Parallel()

	input := strings.NewReader("\n")
	output := &bytes.Buffer{}
	prompter := fix.NewTerminalPrompter(input, output)

	finding := &validation.Error{
		UnderlyingError: errors.New("issue"),
		Node:            &yaml.Node{Line: 1, Column: 1},
		Rule:            "test-rule",
	}
	f := &mockInteractiveFix{
		description: "pick",
		prompts: []validation.Prompt{
			{
				Type:    validation.PromptChoice,
				Message: "Choose:",
				Choices: []string{"a", "b"},
				Default: "a",
			},
		},
	}

	responses, err := prompter.PromptFix(finding, f)
	require.NoError(t, err, "PromptFix should not fail")
	assert.Equal(t, []string{"a"}, responses, "should return default")
}

func TestTerminalPrompter_Choice_Skip(t *testing.T) {
	t.Parallel()

	input := strings.NewReader("s\n")
	output := &bytes.Buffer{}
	prompter := fix.NewTerminalPrompter(input, output)

	finding := &validation.Error{
		UnderlyingError: errors.New("issue"),
		Node:            &yaml.Node{Line: 1, Column: 1},
		Rule:            "test-rule",
	}
	f := &mockInteractiveFix{
		description: "pick",
		prompts: []validation.Prompt{
			{
				Type:    validation.PromptChoice,
				Message: "Choose:",
				Choices: []string{"a", "b"},
			},
		},
	}

	_, err := prompter.PromptFix(finding, f)
	require.Error(t, err, "should return error on skip")
	assert.ErrorIs(t, err, validation.ErrSkipFix, "should return ErrSkipFix")
}

func TestTerminalPrompter_FreeText_Success(t *testing.T) {
	t.Parallel()

	input := strings.NewReader("my description\n")
	output := &bytes.Buffer{}
	prompter := fix.NewTerminalPrompter(input, output)

	finding := &validation.Error{
		UnderlyingError: errors.New("missing description"),
		Node:            &yaml.Node{Line: 5, Column: 1},
		Rule:            "test-rule",
	}
	f := &mockInteractiveFix{
		description: "add description",
		prompts: []validation.Prompt{
			{
				Type:    validation.PromptFreeText,
				Message: "Enter description",
			},
		},
	}

	responses, err := prompter.PromptFix(finding, f)
	require.NoError(t, err, "PromptFix should not fail")
	assert.Equal(t, []string{"my description"}, responses, "should return entered text")
}

func TestTerminalPrompter_FreeText_Skip(t *testing.T) {
	t.Parallel()

	input := strings.NewReader("s\n")
	output := &bytes.Buffer{}
	prompter := fix.NewTerminalPrompter(input, output)

	finding := &validation.Error{
		UnderlyingError: errors.New("issue"),
		Node:            &yaml.Node{Line: 1, Column: 1},
		Rule:            "test-rule",
	}
	f := &mockInteractiveFix{
		description: "add value",
		prompts: []validation.Prompt{
			{
				Type:    validation.PromptFreeText,
				Message: "Enter value",
			},
		},
	}

	_, err := prompter.PromptFix(finding, f)
	require.Error(t, err, "should return error on skip")
	assert.ErrorIs(t, err, validation.ErrSkipFix, "should return ErrSkipFix")
}

func TestTerminalPrompter_Choice_InvalidThenValid(t *testing.T) {
	t.Parallel()

	// First input "abc" is invalid, then "2" is valid
	input := strings.NewReader("abc\n2\n")
	output := &bytes.Buffer{}
	prompter := fix.NewTerminalPrompter(input, output)

	finding := &validation.Error{
		UnderlyingError: errors.New("issue"),
		Node:            &yaml.Node{Line: 1, Column: 1},
		Rule:            "test-rule",
	}
	f := &mockInteractiveFix{
		description: "pick",
		prompts: []validation.Prompt{
			{
				Type:    validation.PromptChoice,
				Message: "Choose:",
				Choices: []string{"a", "b"},
			},
		},
	}

	responses, err := prompter.PromptFix(finding, f)
	require.NoError(t, err, "PromptFix should succeed after re-prompt")
	assert.Equal(t, []string{"b"}, responses, "should return the choice from second attempt")
	assert.Contains(t, output.String(), "Invalid choice", "should show invalid choice message")
}

func TestTerminalPrompter_Choice_OutOfRangeThenValid(t *testing.T) {
	t.Parallel()

	// "99" is out of range, then "1" is valid
	input := strings.NewReader("99\n1\n")
	output := &bytes.Buffer{}
	prompter := fix.NewTerminalPrompter(input, output)

	finding := &validation.Error{
		UnderlyingError: errors.New("issue"),
		Node:            &yaml.Node{Line: 1, Column: 1},
		Rule:            "test-rule",
	}
	f := &mockInteractiveFix{
		description: "pick",
		prompts: []validation.Prompt{
			{
				Type:    validation.PromptChoice,
				Message: "Choose:",
				Choices: []string{"x", "y"},
			},
		},
	}

	responses, err := prompter.PromptFix(finding, f)
	require.NoError(t, err, "PromptFix should succeed after re-prompt")
	assert.Equal(t, []string{"x"}, responses, "should return the choice from second attempt")
	assert.Contains(t, output.String(), "Invalid choice", "should show invalid choice message")
}

func TestTerminalPrompter_Confirm_Yes(t *testing.T) {
	t.Parallel()

	input := strings.NewReader("y\n")
	output := &bytes.Buffer{}
	prompter := fix.NewTerminalPrompter(input, output)

	result, err := prompter.Confirm("Apply fix?")
	require.NoError(t, err, "Confirm should not fail")
	assert.True(t, result, "should return true for 'y'")
}

func TestTerminalPrompter_Confirm_No(t *testing.T) {
	t.Parallel()

	input := strings.NewReader("n\n")
	output := &bytes.Buffer{}
	prompter := fix.NewTerminalPrompter(input, output)

	result, err := prompter.Confirm("Apply fix?")
	require.NoError(t, err, "Confirm should not fail")
	assert.False(t, result, "should return false for 'n'")
}

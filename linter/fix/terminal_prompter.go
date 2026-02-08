package fix

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/speakeasy-api/openapi/validation"
)

// TerminalPrompter implements Prompter using stdin/stdout for terminal interaction.
type TerminalPrompter struct {
	reader *bufio.Reader
	writer io.Writer
}

// NewTerminalPrompter creates a new terminal-based prompter.
func NewTerminalPrompter(in io.Reader, out io.Writer) *TerminalPrompter {
	return &TerminalPrompter{
		reader: bufio.NewReader(in),
		writer: out,
	}
}

// writef writes formatted output to the prompter's writer, ignoring write errors
// since terminal output failures are not recoverable.
func (p *TerminalPrompter) writef(format string, args ...any) {
	_, _ = fmt.Fprintf(p.writer, format, args...)
}

func (p *TerminalPrompter) PromptFix(finding *validation.Error, fix validation.Fix) ([]string, error) {
	// Display context about the error
	p.writef("\n[%d:%d] %s %s\n", finding.GetLineNumber(), finding.GetColumnNumber(), finding.Rule, finding.UnderlyingError.Error())
	p.writef("  Fix: %s\n", fix.Description())

	prompts := fix.Prompts()
	responses := make([]string, len(prompts))

	for i, prompt := range prompts {
		response, err := p.promptOne(prompt)
		if err != nil {
			return nil, err
		}
		responses[i] = response
	}

	return responses, nil
}

func (p *TerminalPrompter) promptOne(prompt validation.Prompt) (string, error) {
	switch prompt.Type {
	case validation.PromptChoice:
		return p.promptChoice(prompt)
	case validation.PromptFreeText:
		return p.promptFreeText(prompt)
	default:
		return "", fmt.Errorf("unknown prompt type: %d", prompt.Type)
	}
}

func (p *TerminalPrompter) promptChoice(prompt validation.Prompt) (string, error) {
	p.writef("  %s\n", prompt.Message)
	for j, choice := range prompt.Choices {
		p.writef("    [%d] %s\n", j+1, choice)
	}
	p.writef("    [s] Skip\n")

	for {
		if prompt.Default != "" {
			p.writef("  (default: %s) > ", prompt.Default)
		} else {
			p.writef("  > ")
		}

		line, err := p.reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("reading input: %w", err)
		}
		line = strings.TrimSpace(line)

		if line == "s" || line == "skip" {
			return "", validation.ErrSkipFix
		}

		if line == "" && prompt.Default != "" {
			return prompt.Default, nil
		}

		idx, err := strconv.Atoi(line)
		if err != nil || idx < 1 || idx > len(prompt.Choices) {
			p.writef("  Invalid choice: %s (enter 1-%d or s to skip)\n", line, len(prompt.Choices))
			continue
		}

		return prompt.Choices[idx-1], nil
	}
}

func (p *TerminalPrompter) promptFreeText(prompt validation.Prompt) (string, error) {
	p.writef("  %s", prompt.Message)
	if prompt.Default != "" {
		p.writef(" (default: %s)", prompt.Default)
	}
	p.writef(" [s to skip]: ")

	line, err := p.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}
	line = strings.TrimSpace(line)

	if line == "s" || line == "skip" {
		return "", validation.ErrSkipFix
	}

	if line == "" && prompt.Default != "" {
		return prompt.Default, nil
	}

	if line == "" {
		return "", validation.ErrSkipFix
	}

	return line, nil
}

func (p *TerminalPrompter) Confirm(message string) (bool, error) {
	p.writef("%s [y/n]: ", message)

	line, err := p.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("reading input: %w", err)
	}
	line = strings.ToLower(strings.TrimSpace(line))

	return line == "y" || line == "yes", nil
}

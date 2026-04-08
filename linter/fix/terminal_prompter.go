package fix

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/speakeasy-api/openapi/validation"
	"golang.org/x/term"
)

// TerminalPrompter implements Prompter using stdin/stdout for terminal interaction.
type TerminalPrompter struct {
	reader *bufio.Reader
	writer io.Writer
	input  io.Reader
}

// NewTerminalPrompter creates a new terminal-based prompter.
func NewTerminalPrompter(in io.Reader, out io.Writer) *TerminalPrompter {
	return &TerminalPrompter{
		reader: bufio.NewReader(in),
		writer: out,
		input:  in,
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
	p.writef("    [r] Skip remaining fixes for this rule\n")
	p.writef("    [e] Exit interactive fixing\n")

	for {
		if prompt.Default != "" {
			p.writef("  (default: %s) > ", prompt.Default)
		} else {
			p.writef("  > ")
		}

		line, err := p.readPromptLine()
		if err != nil {
			if errors.Is(err, validation.ErrExitInteractive) {
				return "", err
			}
			return "", fmt.Errorf("reading input: %w", err)
		}
		line = strings.TrimSpace(line)

		switch strings.ToLower(line) {
		case "s", "skip":
			return "", validation.ErrSkipFix
		case "r":
			return "", validation.ErrSkipRule
		case "e", "exit":
			return "", validation.ErrExitInteractive
		}

		if isEscapeInput(line) {
			return "", validation.ErrExitInteractive
		}

		if line == "" && prompt.Default != "" {
			return prompt.Default, nil
		}

		idx, err := strconv.Atoi(line)
		if err != nil || idx < 1 || idx > len(prompt.Choices) {
			p.writef("  Invalid choice: %s (enter 1-%d, s, r, e, or Escape)\n", line, len(prompt.Choices))
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
	p.writef(" [s=skip, r=skip rule, e=exit; prefix \\ for literal]: ")

	line, err := p.readPromptLine()
	if err != nil {
		if errors.Is(err, validation.ErrExitInteractive) {
			return "", err
		}
		return "", fmt.Errorf("reading input: %w", err)
	}
	line = strings.TrimSpace(line)

	if unescaped, ok := unescapeReservedControlInput(line); ok {
		line = unescaped
	} else {
		switch strings.ToLower(line) {
		case "s", "skip":
			return "", validation.ErrSkipFix
		case "r":
			return "", validation.ErrSkipRule
		case "e", "exit":
			return "", validation.ErrExitInteractive
		}

		if isEscapeInput(line) {
			return "", validation.ErrExitInteractive
		}
	}

	if line == "" && prompt.Default != "" {
		return prompt.Default, nil
	}

	if line == "" {
		return "", validation.ErrSkipFix
	}

	return line, nil
}

func (p *TerminalPrompter) readPromptLine() (string, error) {
	if f, ok := p.input.(*os.File); ok {
		fd := int(f.Fd())
		if term.IsTerminal(fd) {
			state, err := term.MakeRaw(fd)
			if err == nil {
				defer func() {
					_ = term.Restore(fd, state)
				}()

				var b [1]byte
				if _, readErr := syscall.Read(fd, b[:]); readErr != nil {
					return "", readErr
				}

				first := b[0]
				if first == 0x1b {
					return "", validation.ErrExitInteractive
				}

				var sb strings.Builder
				sb.WriteByte(first)
				for first != '\n' {
					if _, readErr := syscall.Read(fd, b[:]); readErr != nil {
						return "", readErr
					}
					next := b[0]
					sb.WriteByte(next)
					first = next
				}
				return sb.String(), nil
			}
		}
	}

	return p.reader.ReadString('\n')
}

func unescapeReservedControlInput(line string) (string, bool) {
	if !strings.HasPrefix(line, "\\") {
		return "", false
	}

	remainder := strings.TrimPrefix(line, "\\")
	switch strings.ToLower(remainder) {
	case "s", "skip", "r", "e", "exit":
		return remainder, true
	}

	if isEscapeInput(remainder) {
		return remainder, true
	}

	return "", false
}

func isEscapeInput(line string) bool {
	if line == "" {
		return false
	}

	r, size := utf8.DecodeRuneInString(line)
	return r == '\x1b' && size == len(line)
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

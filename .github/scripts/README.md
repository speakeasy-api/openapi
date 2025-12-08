# CI Scripts

This directory contains scripts used by GitHub Actions workflows.

## format-coverage.sh

Formats Go coverage output as a markdown table with color-coded indicators.

### Usage

```bash
./format-coverage.sh <coverage-file> [current-coverage] [main-coverage]
```

**Arguments:**
- `coverage-file`: Path to the Go coverage file (typically `coverage.out`)
- `current-coverage`: (optional) Overall coverage percentage for display (e.g., "74.3%")
- `main-coverage`: (optional) Main branch coverage percentage for comparison (e.g., "74.0%")

### Examples

**Basic usage:**
```bash
# Generate coverage file first
mise test-coverage

# Format the coverage report
./.github/scripts/format-coverage.sh coverage.out
```

**With coverage percentages:**
```bash
# Calculate coverage
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')

# Format with coverage display
./.github/scripts/format-coverage.sh coverage.out "$COVERAGE"
```

**With comparison to main:**
```bash
./.github/scripts/format-coverage.sh coverage.out "75.2%" "74.3%"
```

### Output Format

The script generates a markdown report with:
- Overall coverage statistics
- Coverage comparison (if main branch coverage provided)
- Table of coverage by package with color-coded indicators:
  - 🟢 Green: ≥90% coverage
  - 🟡 Yellow: ≥75% coverage
  - 🟠 Orange: ≥50% coverage
  - 🔴 Red: <50% coverage
- Collapsible detailed coverage by function

### Local Testing

To test the output locally:

```bash
# Run tests with coverage
mise test-coverage

# Format and preview the output
./.github/scripts/format-coverage.sh coverage.out "74.3%" "74.3%" | less
```

Or save to a file for inspection:

```bash
./.github/scripts/format-coverage.sh coverage.out "74.3%" > coverage-report.md
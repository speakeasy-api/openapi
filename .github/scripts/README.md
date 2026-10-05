# CI Scripts

This directory contains scripts used by GitHub Actions workflows.

## format-coverage.sh

Formats Go statement coverage as a markdown table with color-coded indicators. Uses `normalize-coverage.sh` to filter and merge the profile before calculating package totals. Codecov measures line coverage separately, so its percentage can differ.

### Usage

```bash
./format-coverage.sh <coverage-file> [current-coverage] [main-coverage] [main-status]
```

**Arguments:**
- `coverage-file`: Path to the Go coverage file (typically `coverage.out`)
- `current-coverage`: (optional) Overall statement coverage percentage for display (e.g., "89.2%"), calculated from the normalised profile
- `main-coverage`: (optional) Main branch statement coverage percentage for comparison (e.g., "85.3%"), calculated from the same normalised scope
- `main-status`: (optional) Reason main coverage is unavailable (e.g., "tests timed out after 900 seconds"). Pass an empty `main-coverage` argument to display this reason and omit the comparison, rather than treating a failed measurement as 0%.

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

**When main coverage is unavailable:**
```bash
./.github/scripts/format-coverage.sh coverage.out "89.2%" "" "tests timed out after 900 seconds"
```

### Output Format

The script generates a markdown report with:
- Overall coverage statistics
- Coverage comparison (if main branch coverage provided), or an unavailable-baseline reason when `main-status` is provided
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
```

## normalize-coverage.sh

Filters `/cmd/` and `/tests/` files from a Go coverage profile and merges repeated blocks from cross-package instrumentation by summing their hit counts. The coverage task, formatter, and main-baseline workflow share this helper so they measure the same library scope.

```bash
bash ./.github/scripts/normalize-coverage.sh coverage.out > coverage.normalized.out
```

Use a different output file to avoid truncating the input profile. Calculate statement coverage from the normalised profile:

```bash
go tool cover -func=coverage.normalized.out | awk '/^total:/ {print $NF}'
```
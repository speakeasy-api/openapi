package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCase represents a single test case from the JSON Schema Test Suite
type TestCase struct {
	Description string          `json:"description"`
	Comment     string          `json:"comment,omitempty"`
	Schema      json.RawMessage `json:"schema"`
	Tests       []Test          `json:"tests"`
}

// Test represents a single test within a test case
type Test struct {
	Description string      `json:"description"`
	Comment     string      `json:"comment,omitempty"`
	Data        interface{} `json:"data"`
	Valid       bool        `json:"valid"`
}

// Blacklisted test files that we don't support or want to skip
// TODO work on improving support for these files
var blacklistedFiles = map[string]string{
	// Anchor resolution edge cases
	"optional/anchor.json": "contains edge cases for anchor resolution",
	"anchor.json":          "contains edge cases for anchor resolution",

	// Unknown keyword and ID edge cases
	"optional/unknownKeyword.json": "contains edge cases for unknown keyword handling",
	"optional/id.json":             "contains edge cases for ID resolution",
}

// Blacklisted test cases within specific files
// Key format: "filename:case_number"
// TODO work on improving support for these test cases
var blacklistedTestCases = map[string]string{
	// Remote reference tests that require $id base URI change support
	"refRemote.json:2":  "requires anchor resolution support",
	"refRemote.json:4":  "requires $id base URI change support",
	"refRemote.json:5":  "requires $id base URI change support",
	"refRemote.json:6":  "requires $id base URI change support",
	"refRemote.json:7":  "requires $id base URI change support",
	"refRemote.json:8":  "requires external reference resolution",
	"refRemote.json:9":  "requires anchor resolution support",
	"refRemote.json:10": "requires $id base URI change support",
	"refRemote.json:13": "requires nested absolute reference resolution",
	"refRemote.json:14": "requires detached anchor resolution support",

	// ref.json tests that require advanced reference resolution features
	"ref.json:11": "requires external reference resolution with $id",
	"ref.json:15": "requires relative URI resolution with $id",
	"ref.json:16": "requires absolute URI resolution with $id",
	"ref.json:17": "requires complex $id resolution chain",
	"ref.json:18": "requires $id evaluation before $ref",
	"ref.json:19": "requires $id and $anchor evaluation before $ref",
	"ref.json:20": "requires URN scheme support",
	"ref.json:25": "requires URN scheme with JSON pointer",
	"ref.json:26": "requires URN scheme with anchor",
	"ref.json:27": "requires URN scheme with nested references",
	"ref.json:28": "requires conditional schema reference resolution",
	"ref.json:29": "requires conditional schema reference resolution",
	"ref.json:30": "requires conditional schema reference resolution",
	"ref.json:31": "requires absolute path reference resolution",

	// dynamicRef.json tests - all failing due to lack of dynamic reference support
	"dynamicRef.json:0":  "requires dynamic reference resolution support",
	"dynamicRef.json:2":  "requires dynamic reference resolution support",
	"dynamicRef.json:3":  "requires dynamic reference resolution support",
	"dynamicRef.json:4":  "requires dynamic reference resolution support",
	"dynamicRef.json:5":  "requires dynamic reference resolution support",
	"dynamicRef.json:6":  "requires dynamic reference resolution support",
	"dynamicRef.json:7":  "requires dynamic reference resolution support",
	"dynamicRef.json:8":  "requires dynamic reference resolution support",
	"dynamicRef.json:9":  "requires dynamic reference resolution support",
	"dynamicRef.json:10": "requires dynamic reference resolution support",
	"dynamicRef.json:11": "requires dynamic reference resolution support",
	"dynamicRef.json:12": "requires dynamic reference resolution support",
	"dynamicRef.json:13": "requires dynamic reference resolution support",
	"dynamicRef.json:14": "requires dynamic reference resolution support",
	"dynamicRef.json:15": "requires dynamic reference resolution support",
	"dynamicRef.json:16": "requires dynamic reference resolution support",
	"dynamicRef.json:19": "requires dynamic reference resolution support",

	// optional/dynamicRef.json tests
	"optional/dynamicRef.json:0": "requires dynamic reference resolution support",

	// unevaluatedItems.json tests with dynamicRef
	"unevaluatedItems.json:18": "requires dynamic reference resolution support",

	// unevaluatedProperties.json tests with dynamicRef
	"unevaluatedProperties.json:21": "requires dynamic reference resolution support",
}

const testSuiteDir = "testsuite/tests/draft2020-12"

// Global variable to hold the remote server instance
var remoteServer *RemoteServer

// Thread-safe coverage tracking
type CoverageTracker struct {
	totalFiles   int64
	skippedFiles int64
	totalCases   int64
	skippedCases int64
	passedCases  int64
	failedCases  int64
}

func (c *CoverageTracker) AddFile() {
	atomic.AddInt64(&c.totalFiles, 1)
}

func (c *CoverageTracker) AddSkippedFile() {
	atomic.AddInt64(&c.skippedFiles, 1)
}

func (c *CoverageTracker) AddCases(total, skipped, passed, failed int64) {
	atomic.AddInt64(&c.totalCases, total)
	atomic.AddInt64(&c.skippedCases, skipped)
	atomic.AddInt64(&c.passedCases, passed)
	atomic.AddInt64(&c.failedCases, failed)
}

func (c *CoverageTracker) GetStats() (int, int, int, int, int, int) {
	return int(atomic.LoadInt64(&c.totalFiles)),
		int(atomic.LoadInt64(&c.skippedFiles)),
		int(atomic.LoadInt64(&c.totalCases)),
		int(atomic.LoadInt64(&c.skippedCases)),
		int(atomic.LoadInt64(&c.passedCases)),
		int(atomic.LoadInt64(&c.failedCases))
}

func TestMain(m *testing.M) {
	// Check if the git submodule is initialized
	if !isSubmoduleInitialized(testSuiteDir) {
		log.Println("JSON Schema Test Suite submodule not initialized. Run 'git submodule update --init --recursive' to enable these tests.")
		return
	}

	// Start the remote server for remote reference tests
	var err error
	remoteServer, err = startRemoteServer()
	if err != nil {
		log.Printf("Warning: Failed to start remote server for remote reference tests: %v", err)
		log.Println("Remote reference tests will be skipped.")
	} else {
		log.Printf("Remote server started at %s", remoteServer.GetBaseURL())
	}

	// Run tests
	exitCode := m.Run()

	// Clean up the remote server
	if remoteServer != nil {
		remoteServer.Stop()
	}

	os.Exit(exitCode)
}

// TestJSONSchemaTestSuite_RoundTrip runs the JSON Schema Test Suite tests
// focusing on schema parsing, validation, reference resolution, and round-trip marshalling
func TestJSONSchemaTestSuite_RoundTrip(t *testing.T) {
	t.Parallel()

	// Get all test files
	testFiles := getAllTestFiles(t, testSuiteDir)

	// Thread-safe coverage tracking
	tracker := &CoverageTracker{}
	var wg sync.WaitGroup

	for _, testFile := range testFiles {
		tracker.AddFile()

		// Check if this file is blacklisted
		if reason, isBlacklisted := blacklistedFiles[testFile]; isBlacklisted {
			tracker.AddSkippedFile()
			t.Run(testFile, func(t *testing.T) {
				t.Skipf("Skipping blacklisted file: %s", reason)
			})
			continue
		}

		wg.Add(1)
		t.Run(testFile, func(t *testing.T) {
			defer wg.Done()
			t.Parallel()
			fileCases, fileSkipped, filePassed, fileFailed := runRoundTripTestFile(t, filepath.Join(testSuiteDir, testFile))
			tracker.AddCases(int64(fileCases), int64(fileSkipped), int64(filePassed), int64(fileFailed))
		})
	}

	// Print coverage summary after all tests complete
	t.Cleanup(func() {
		wg.Wait() // Wait for all parallel tests to complete
		totalFiles, skippedFiles, totalCases, skippedCases, passedCases, failedCases := tracker.GetStats()
		printCoverageSummary(t, totalFiles, skippedFiles, totalCases, skippedCases, passedCases, failedCases)
	})
}

// getAllTestFiles returns all JSON test files in the test suite directory
func getAllTestFiles(t *testing.T, testSuiteDir string) []string {
	t.Helper()
	var testFiles []string

	err := filepath.WalkDir(testSuiteDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".json") {
			// Get relative path from testSuiteDir
			relPath, err := filepath.Rel(testSuiteDir, path)
			if err != nil {
				return err
			}
			testFiles = append(testFiles, relPath)
		}

		return nil
	})

	require.NoError(t, err)
	return testFiles
}

// runRoundTripTestFile runs all test cases in a single test file with round-trip testing
// Returns: totalCases, skippedCases, passedCases, failedCases
func runRoundTripTestFile(t *testing.T, testFilePath string) (int, int, int, int) {
	t.Helper()
	// Read the test file
	data, err := os.ReadFile(testFilePath)
	require.NoError(t, err, "failed to read test file: %s", testFilePath)

	// Parse the test cases
	var testCases []TestCase
	err = json.Unmarshal(data, &testCases)
	require.NoError(t, err, "failed to parse test file: %s", testFilePath)

	var skippedCases, passedCases, failedCases int

	// Run each test case
	for i, testCase := range testCases {
		testPassed := t.Run(fmt.Sprintf("case_%d_%s", i, sanitizeTestName(testCase.Description)), func(t *testing.T) {
			// Check if this specific test case is blacklisted
			fileName := filepath.Base(testFilePath)
			testCaseKey := fmt.Sprintf("%s:%d", fileName, i)
			if reason, isBlacklisted := blacklistedTestCases[testCaseKey]; isBlacklisted {
				skippedCases++
				t.Skipf("Skipping blacklisted test case: %s", reason)
				return
			}

			runRoundTripTestCase(t, testCase, testFilePath)
		})

		if testPassed {
			passedCases++
		} else {
			failedCases++
		}
	}

	return len(testCases), skippedCases, passedCases, failedCases
}

// runRoundTripTestCase runs a single test case with round-trip testing
func runRoundTripTestCase(t *testing.T, testCase TestCase, testFilePath string) {
	t.Helper()
	ctx := t.Context()

	// Step 1: Unmarshal the schema
	var schema oas3.JSONSchema[oas3.Referenceable]
	validationErrs, err := marshaller.Unmarshal(ctx, bytes.NewReader(testCase.Schema), &schema)
	require.NoError(t, err, "failed to unmarshal schema for test case: %s", testCase.Description)
	require.Empty(t, validationErrs, "schema validation errors for test case: %s", testCase.Description)

	// Step 2: Run validation on the schema
	schemaValidationErrs := schema.Validate(ctx)
	require.Empty(t, schemaValidationErrs, "schema validation failed for test case: %s", testCase.Description)

	// Step 4: Use the Walk API to walk through the schema and resolve any references
	err = walkAndResolveReferences(ctx, t, &schema, testFilePath)
	require.NoError(t, err, "failed to walk and resolve references for test case: %s", testCase.Description)

	// Step 5: Marshal the schema back to JSON
	var buf bytes.Buffer
	err = marshaller.Marshal(ctx, &schema, &buf)
	require.NoError(t, err, "failed to marshal schema for test case: %s", testCase.Description)

	// Step 6: Use assert.JSONEq to check the schema makes a successful round trip
	originalJSON := string(testCase.Schema)
	roundTripJSON := buf.String()

	// For round-trip comparison, we need to normalize both JSONs since the order might differ
	// and some fields might be added/removed during the process
	assert.JSONEq(t, originalJSON, roundTripJSON, "schema round-trip failed for test case: %s", testCase.Description)

	// Log success information
	t.Logf("✅ Round-trip successful for test case: %s", testCase.Description)
	t.Logf("   Original schema size: %d bytes", len(originalJSON))
	t.Logf("   Round-trip schema size: %d bytes", len(roundTripJSON))
}

// walkAndResolveReferences walks through the schema using the Walk API and resolves any references
func walkAndResolveReferences(ctx context.Context, t *testing.T, schema *oas3.JSONSchema[oas3.Referenceable], testFilePath string) error {
	t.Helper()
	if schema == nil {
		return nil
	}

	// Walk through the schema and resolve any references we find
	for item := range oas3.Walk(ctx, schema) {
		err := item.Match(oas3.SchemaMatcher{
			Schema: func(s *oas3.JSONSchema[oas3.Referenceable]) error {
				// If this is a reference, try to resolve it
				if s.IsReference() {
					// Create resolve options
					resolveOpts := oas3.ResolveOptions{
						TargetLocation: testFilePath,
						RootDocument:   schema,
					}

					// If we have a remote server running, use its custom HTTP client
					if remoteServer != nil {
						resolveOpts.HTTPClient = remoteServer.GetHTTPClient()
					}

					// Attempt to resolve the reference
					// Most test suite schemas should have resolvable references within the schema
					vErrs, resolveErr := s.Resolve(ctx, resolveOpts)

					assert.NoError(t, resolveErr)
					assert.Empty(t, vErrs)
				}
				return nil
			},
		})
		if err != nil {
			return fmt.Errorf("failed to process schema during walk: %w", err)
		}
	}

	return nil
}

// sanitizeTestName sanitizes a test name for use as a Go test name
func sanitizeTestName(name string) string {
	// Replace spaces and special characters with underscores
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, "(", "_")
	name = strings.ReplaceAll(name, ")", "_")
	name = strings.ReplaceAll(name, "[", "_")
	name = strings.ReplaceAll(name, "]", "_")
	name = strings.ReplaceAll(name, "{", "_")
	name = strings.ReplaceAll(name, "}", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, ",", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, ";", "_")
	name = strings.ReplaceAll(name, "!", "_")
	name = strings.ReplaceAll(name, "?", "_")
	name = strings.ReplaceAll(name, "'", "_")
	name = strings.ReplaceAll(name, "\"", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "<", "_")
	name = strings.ReplaceAll(name, ">", "_")
	name = strings.ReplaceAll(name, "=", "_")
	name = strings.ReplaceAll(name, "+", "_")
	name = strings.ReplaceAll(name, "*", "_")
	name = strings.ReplaceAll(name, "&", "_")
	name = strings.ReplaceAll(name, "%", "_")
	name = strings.ReplaceAll(name, "$", "_")
	name = strings.ReplaceAll(name, "#", "_")
	name = strings.ReplaceAll(name, "@", "_")

	// Remove multiple consecutive underscores
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}

	// Trim leading and trailing underscores
	name = strings.Trim(name, "_")

	// Ensure the name is not empty
	if name == "" {
		name = "unnamed_test"
	}

	return name
}

// isSubmoduleInitialized checks if the git submodule is properly initialized
func isSubmoduleInitialized(testSuiteDir string) bool {
	// Check if the directory exists and contains expected files
	if _, err := os.Stat(testSuiteDir); os.IsNotExist(err) {
		return false
	}

	// Check if the directory contains test files (should have .json files)
	entries, err := os.ReadDir(testSuiteDir)
	if err != nil {
		return false
	}

	// Look for at least one .json file to confirm the submodule is initialized
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			return true
		}
	}

	return false
}

// printCoverageSummary prints a summary of test coverage statistics
func printCoverageSummary(t *testing.T, totalFiles, skippedFiles, totalCases, skippedCases, passedCases, failedCases int) {
	t.Helper()

	runFiles := totalFiles - skippedFiles
	filesCoverage := float64(runFiles) / float64(totalFiles) * 100

	casesCoverage := float64(passedCases) / float64(totalCases) * 100

	t.Logf("\n"+
		"📊 JSON Schema Test Suite Coverage Summary\n"+
		"==========================================\n"+
		"Files:      %d/%d (%.1f%%) - %d skipped\n"+
		"Test Cases: %d/%d (%.1f%%) - %d skipped\n"+
		"Status:     %d passed, %d failed, %d skipped, %d total\n",
		runFiles, totalFiles, filesCoverage, skippedFiles,
		passedCases, totalCases, casesCoverage, skippedCases,
		passedCases, failedCases, skippedCases, totalCases)
}

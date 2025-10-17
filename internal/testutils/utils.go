package testutils

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"iter"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// TODO use these more in tests
func CreateStringYamlNode(value string, line, column int) *yaml.Node {
	return &yaml.Node{
		Value:  value,
		Kind:   yaml.ScalarNode,
		Tag:    "!!str",
		Line:   line,
		Column: column,
	}
}

func CreateIntYamlNode(value int, line, column int) *yaml.Node {
	return &yaml.Node{
		Value:  strconv.Itoa(value),
		Kind:   yaml.ScalarNode,
		Tag:    "!!int",
		Line:   line,
		Column: column,
	}
}

func CreateBoolYamlNode(value bool, line, column int) *yaml.Node {
	return &yaml.Node{
		Value:  strconv.FormatBool(value),
		Kind:   yaml.ScalarNode,
		Tag:    "!!bool",
		Line:   line,
		Column: column,
	}
}

func CreateMapYamlNode(contents []*yaml.Node, line, column int) *yaml.Node {
	return &yaml.Node{
		Content: contents,
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Line:    line,
		Column:  column,
	}
}

type SequencedMap interface {
	Len() int
	AllUntyped() iter.Seq2[any, any]
	GetUntyped(key any) (any, bool)
}

// isInterfaceNil checks if an interface has a nil underlying value
func isInterfaceNil(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		return false
	}
}

func AssertEqualSequencedMap(t *testing.T, expected, actual SequencedMap) {
	t.Helper()
	// Check if both are truly nil (interface with nil type and value)
	if expected == nil && actual == nil {
		return
	}

	// Check if either is nil or has a nil underlying value
	expectedIsNil := expected == nil || (expected != nil && isInterfaceNil(expected))
	actualIsNil := actual == nil || (actual != nil && isInterfaceNil(actual))

	if expectedIsNil && actualIsNil {
		return
	}

	if expectedIsNil || actualIsNil {
		assert.Fail(t, "expected and actual must not be nil")
		return
	}

	assert.EqualExportedValues(t, expected, actual)
	assert.Equal(t, expected.Len(), actual.Len())

	alreadySeen := map[any]bool{}

	for k, v := range expected.AllUntyped() {
		actualV, ok := actual.GetUntyped(k)
		assert.True(t, ok)
		assert.EqualExportedValues(t, v, actualV)

		alreadySeen[k] = true
	}

	for k, v := range actual.AllUntyped() {
		if _, ok := alreadySeen[k]; ok {
			continue
		}

		actualV, ok := actual.GetUntyped(k)
		assert.True(t, ok)
		assert.EqualExportedValues(t, v, actualV)
	}
}

// DownloadFile downloads a file from a URL and caches it to avoid re-downloading.
// Uses the provided cacheEnvVar for cache location, fallback to system temp dir.
// The cacheDirName is used as the subdirectory name under the cache directory.
func DownloadFile(url, cacheEnvVar, cacheDirName string) (io.ReadCloser, error) {
	// Use environment variable for cache directory, fallback to system temp dir
	cacheDir := os.Getenv(cacheEnvVar)
	if cacheDir == "" {
		cacheDir = os.TempDir()
	}
	tempDir := filepath.Join(cacheDir, cacheDirName)

	if err := os.MkdirAll(tempDir, 0o750); err != nil {
		return nil, err
	}

	// hash url to create a unique filename
	hash := sha256.Sum256([]byte(url))
	filename := hex.EncodeToString(hash[:])

	filepath := filepath.Join(tempDir, filename)

	// check if file exists and return it otherwise download it
	r, err := os.Open(filepath) // #nosec G304 -- filepath is controlled by caller in tests
	if err == nil {
		return r, nil
	}

	resp, err := http.Get(url) // #nosec G107 -- url is controlled by caller in tests
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, io.ErrUnexpectedEOF
	}
	defer resp.Body.Close()

	// Read all data from response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Write data to cache file
	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY, 0o600) // #nosec G304 -- filepath is controlled by caller in tests
	if err != nil {
		return nil, err
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return nil, err
	}

	// Return the data as a ReadCloser
	return io.NopCloser(bytes.NewReader(data)), nil
}

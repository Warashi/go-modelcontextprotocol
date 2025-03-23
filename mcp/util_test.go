package mcp

import (
	"encoding/json"
	"io"
	"reflect"
	"testing"
)

// assertJSONEqual checks if two JSON strings are equal.
func assertJSONEqual(t *testing.T, expected, actual string) {
	t.Helper()

	var expectedJSON, actualJSON any

	if err := json.Unmarshal([]byte(expected), &expectedJSON); err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %s", err)
	}

	if err := json.Unmarshal([]byte(actual), &actualJSON); err != nil {
		t.Fatalf("Failed to unmarshal actual JSON: %s", err)
	}

	if !reflect.DeepEqual(expectedJSON, actualJSON) {
		t.Errorf("JSON not equal.\nExpected: %s\nActual: %s", expected, actual)
	}
}

// nopReadWriter is a no-op implementation of io.ReadWriter for testing.
type nopReadWriter struct{}

func (nopReadWriter) Read(p []byte) (n int, err error)  { return 0, io.EOF }
func (nopReadWriter) Write(p []byte) (n int, err error) { return len(p), nil }

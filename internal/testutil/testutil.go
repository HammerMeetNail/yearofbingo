// Package testutil provides testing utilities and helpers.
package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// AssertEqual compares two values and fails the test if they're not equal.
func AssertEqual(t *testing.T, expected, actual interface{}, msg string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// AssertNotEqual compares two values and fails the test if they're equal.
func AssertNotEqual(t *testing.T, notExpected, actual interface{}, msg string) {
	t.Helper()
	if notExpected == actual {
		t.Errorf("%s: expected value to not equal %v", msg, notExpected)
	}
}

// AssertNil fails the test if the value is not nil.
func AssertNil(t *testing.T, value interface{}, msg string) {
	t.Helper()
	if value != nil {
		t.Errorf("%s: expected nil, got %v", msg, value)
	}
}

// AssertNotNil fails the test if the value is nil.
func AssertNotNil(t *testing.T, value interface{}, msg string) {
	t.Helper()
	if value == nil {
		t.Errorf("%s: expected non-nil value", msg)
	}
}

// AssertTrue fails the test if the value is not true.
func AssertTrue(t *testing.T, value bool, msg string) {
	t.Helper()
	if !value {
		t.Errorf("%s: expected true", msg)
	}
}

// AssertFalse fails the test if the value is not false.
func AssertFalse(t *testing.T, value bool, msg string) {
	t.Helper()
	if value {
		t.Errorf("%s: expected false", msg)
	}
}

// AssertNoError fails the test if err is not nil.
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Errorf("%s: unexpected error: %v", msg, err)
	}
}

// AssertError fails the test if err is nil.
func AssertError(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: expected error, got nil", msg)
	}
}

// AssertContains fails the test if s does not contain substr.
func AssertContains(t *testing.T, s, substr, msg string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("%s: expected %q to contain %q", msg, s, substr)
	}
}

// AssertStatusCode checks if the response has the expected status code.
func AssertStatusCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if rr.Code != expected {
		t.Errorf("expected status %d, got %d. Body: %s", expected, rr.Code, rr.Body.String())
	}
}

// AssertJSONContains checks if the JSON response contains expected key-value pairs.
func AssertJSONContains(t *testing.T, body []byte, key string, expected interface{}) {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if result[key] != expected {
		t.Errorf("expected %s to be %v, got %v", key, expected, result[key])
	}
}

// NewTestRequest creates a new HTTP request for testing.
func NewTestRequest(method, path string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// NewTestRequestWithJSON creates a new HTTP request with JSON body.
func NewTestRequestWithJSON(t *testing.T, method, path string, data interface{}) *http.Request {
	t.Helper()
	body, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return NewTestRequest(method, path, strings.NewReader(string(body)))
}

// RandomUUID generates a random UUID for testing.
func RandomUUID() uuid.UUID {
	return uuid.New()
}

// RandomEmail generates a random email for testing.
func RandomEmail() string {
	return uuid.New().String()[:8] + "@test.com"
}

// ParseJSONResponse parses a JSON response body into a map.
func ParseJSONResponse(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}
	return result
}

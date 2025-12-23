// Package testutil provides common test utilities for handler and integration tests.
package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NewJSONRequest creates an HTTP request with JSON body.
// The body is marshaled to JSON automatically.
func NewJSONRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		require.NoError(t, err, "failed to marshal request body")
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// NewRequest creates a simple HTTP request without a body.
func NewRequest(t *testing.T, method, path string) *http.Request {
	t.Helper()
	return httptest.NewRequest(method, path, nil)
}

// NewRequestWithBody creates an HTTP request with a string body.
func NewRequestWithBody(t *testing.T, method, path string, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// DoRequest executes a request against a handler and returns the recorder.
func DoRequest(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// MustMarshal marshals a value to JSON string, failing the test on error.
func MustMarshal(t *testing.T, v any) string {
	t.Helper()
	body, err := json.Marshal(v)
	require.NoError(t, err, "failed to marshal value")
	return string(body)
}

// ReadBody reads the response body as bytes.
func ReadBody(t *testing.T, rr *httptest.ResponseRecorder) []byte {
	t.Helper()
	body, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "failed to read response body")
	return body
}

// UnmarshalResponse unmarshals the response body into the target struct.
func UnmarshalResponse[T any](t *testing.T, rr *httptest.ResponseRecorder) *T {
	t.Helper()
	body := ReadBody(t, rr)
	var result T
	require.NoError(t, json.Unmarshal(body, &result), "failed to unmarshal response")
	return &result
}

// UnmarshalErrorResponse unmarshals the response body as an error response.
func UnmarshalErrorResponse(t *testing.T, rr *httptest.ResponseRecorder) map[string]string {
	t.Helper()
	body := ReadBody(t, rr)
	var result map[string]string
	require.NoError(t, json.Unmarshal(body, &result), "failed to unmarshal error response")
	return result
}

// AssertStatus asserts the response status code matches expected.
func AssertStatus(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	assert.Equal(t, expected, rr.Code, "unexpected status code")
}

// AssertStatusOK asserts the response status is 200 OK.
func AssertStatusOK(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	AssertStatus(t, rr, http.StatusOK)
}

// AssertErrorCode asserts the response contains the expected error code.
func AssertErrorCode(t *testing.T, rr *httptest.ResponseRecorder, expectedCode string) {
	t.Helper()
	errResp := UnmarshalErrorResponse(t, rr)
	assert.Equal(t, expectedCode, errResp["error"], "unexpected error code")
}

// AssertStatusAndError asserts both status code and error code.
func AssertStatusAndError(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedCode string) {
	t.Helper()
	AssertStatus(t, rr, expectedStatus)
	AssertErrorCode(t, rr, expectedCode)
}

// AssertJSONContains asserts the response JSON contains the expected key-value pair.
func AssertJSONContains(t *testing.T, rr *httptest.ResponseRecorder, key string, expectedValue any) {
	t.Helper()
	body := ReadBody(t, rr)
	var result map[string]any
	require.NoError(t, json.Unmarshal(body, &result), "failed to unmarshal response")
	assert.Equal(t, expectedValue, result[key], "unexpected value for key %q", key)
}

// AssertJSONHasKey asserts the response JSON contains the specified key.
func AssertJSONHasKey(t *testing.T, rr *httptest.ResponseRecorder, key string) {
	t.Helper()
	body := ReadBody(t, rr)
	var result map[string]any
	require.NoError(t, json.Unmarshal(body, &result), "failed to unmarshal response")
	_, ok := result[key]
	assert.True(t, ok, "expected key %q not found in response", key)
}

package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	dErrors "credo/pkg/domain-errors"
)

func TestWriteError(t *testing.T) {
	t.Run("internal error omits description", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteError(w, dErrors.New(dErrors.CodeInternal, "db failed"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}

		var body map[string]string
		if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body["error"] != "internal_error" {
			t.Fatalf("expected error code internal_error, got %q", body["error"])
		}
		if _, ok := body["error_description"]; ok {
			t.Fatalf("expected error_description to be omitted for internal errors")
		}
	})

	t.Run("bad request includes description", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteError(w, dErrors.New(dErrors.CodeBadRequest, "invalid input"))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}

		var body map[string]string
		if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body["error"] != "bad_request" {
			t.Fatalf("expected error code bad_request, got %q", body["error"])
		}
		if body["error_description"] != "invalid input" {
			t.Fatalf("expected error_description to be returned for bad request")
		}
	})
}

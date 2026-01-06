// Package version provides middleware for API version extraction and validation.
package version

import (
	"encoding/json"
	"net/http"

	id "credo/pkg/domain"
	"credo/pkg/requestcontext"
)

// ExtractVersion creates middleware that extracts the API version from a Chi subrouter.
// When using Chi's r.Route("/v1", ...), the version is already determined by the route match.
// This middleware sets the version in the context for downstream handlers.
//
// Usage:
//
//	r.Route("/v1", func(v1 chi.Router) {
//	    v1.Use(version.ExtractVersion(id.APIVersionV1))
//	    // ... routes
//	})
func ExtractVersion(version id.APIVersion) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := requestcontext.WithAPIVersion(r.Context(), version)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// versionErrorResponse represents the JSON error response for version-related errors.
type versionErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// writeVersionError writes a JSON error response for version-related errors.
func writeVersionError(w http.ResponseWriter, statusCode int, errCode, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := versionErrorResponse{
		Error:            errCode,
		ErrorDescription: description,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

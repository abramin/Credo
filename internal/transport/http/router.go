package httptransport

import (
	"encoding/json"
	"net/http"

	httpErrors "id-gateway/pkg/http-errors"
)

// Handler is the thin HTTP layer. It should delegate to domain services without
// embedding business logic so transport concerns remain isolated.
type Handler struct {
	regulatedMode bool
}

func NewHandler(regulatedMode bool) *Handler {
	return &Handler{regulatedMode: regulatedMode}
}

// NewRouter wires all public endpoints. Each route enforces HTTP verb checks and
// returns a JSON stub until real handlers are implemented.
func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/vc/issue", method(http.MethodPost, h.handleVCIssue))
	mux.HandleFunc("/vc/verify", method(http.MethodPost, h.handleVCVerify))

	mux.HandleFunc("/registry/citizen", method(http.MethodPost, h.handleRegistryCitizen))
	mux.HandleFunc("/registry/sanctions", method(http.MethodPost, h.handleRegistrySanctions))

	mux.HandleFunc("/decision/evaluate", method(http.MethodPost, h.handleDecisionEvaluate))
	mux.HandleFunc("/me/data-export", method(http.MethodGet, h.handleDataExport))
	mux.HandleFunc("/me", method(http.MethodDelete, h.handleDataDeletion))
	return mux
}

func method(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		next(w, r)
	}
}

func (h *Handler) notImplemented(w http.ResponseWriter, endpoint string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message":  "TODO: implement handler",
		"endpoint": endpoint,
	})
}

// writeError centralizes domain error translation to HTTP responses for future
// handlers. Keeping it here ensures consistent JSON error envelopes.
func writeError(w http.ResponseWriter, err error) {
	gw, ok := err.(httpErrors.GatewayError)
	status := http.StatusInternalServerError
	code := string(httpErrors.CodeInternal)
	if ok {
		status = httpErrors.ToHTTPStatus(gw.Code)
		code = string(gw.Code)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": code,
	})
}

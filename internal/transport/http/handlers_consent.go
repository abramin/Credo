package httptransport

import "net/http"

func (h *Handler) handleConsent(w http.ResponseWriter, r *http.Request) {
	h.notImplemented(w, "/auth/consent")
}

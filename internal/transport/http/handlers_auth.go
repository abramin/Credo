package httptransport

import "net/http"

func (h *Handler) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	h.notImplemented(w, "/auth/authorize")
}

func (h *Handler) handleToken(w http.ResponseWriter, r *http.Request) {
	h.notImplemented(w, "/auth/token")
}

func (h *Handler) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	h.notImplemented(w, "/auth/userinfo")
}

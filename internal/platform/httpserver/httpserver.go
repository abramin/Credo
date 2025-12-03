package httpserver

import (
	"net/http"
	"time"
)

// New builds an HTTP server with sane defaults for this project.
func New(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

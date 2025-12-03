package logger

import (
	"log"
	"os"
)

// New returns a basic stdout logger; swap in structured logging when needed.
func New() *log.Logger {
	return log.New(os.Stdout, "", log.LstdFlags)
}

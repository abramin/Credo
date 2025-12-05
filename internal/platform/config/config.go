package config

import (
	"os"
	"time"
)

// Server captures HTTP server level configuration.
type Server struct {
	Addr          string
	RegulatedMode bool
	JWTSigningKey string
}

// RegistryCacheTTL enforces retention for sensitive registry data.
var RegistryCacheTTL = 5 * time.Minute

// FromEnv builds a Server config from environment variables so main stays lean.
func FromEnv() Server {
	addr := os.Getenv("ID_GATEWAY_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	regulated := os.Getenv("REGULATED_MODE") == "true"

	jwtSigningKey := os.Getenv("JWT_SIGNING_KEY")
	if jwtSigningKey == "" {
		// Use a default for development - should be overridden in production
		jwtSigningKey = "dev-secret-key-change-in-production"
	}

	return Server{
		Addr:          addr,
		RegulatedMode: regulated,
		JWTSigningKey: jwtSigningKey,
	}
}

package revocation

import (
	"fmt"
	"time"

	"credo/pkg/platform/sentinel"
)

func validateTTL(ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("ttl must be positive: %w", sentinel.ErrInvalidState)
	}
	return nil
}

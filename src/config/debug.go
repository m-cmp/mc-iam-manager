package config

import (
	"log"
	"os"
	"strconv"
)

// DebugEnabled reports whether echo debug mode should be on.
// When enabled, HTTP error responses include the internal error message,
// so it must stay off outside local development.
func DebugEnabled() bool {
	raw := os.Getenv("MC_IAM_MANAGER_DEBUG")
	if raw == "" {
		return false
	}
	enabled, err := strconv.ParseBool(raw)
	if err != nil {
		log.Printf("[WARN] invalid MC_IAM_MANAGER_DEBUG=%q, using default false", raw)
		return false
	}
	return enabled
}

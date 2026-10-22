package config

import (
	"log"
	"os"
	"strconv"
)

// SwaggerEnabled reports whether the Swagger UI (/swagger/*) is served without authentication.
// The API spec is already published online, so it defaults to true;
// set MC_IAM_MANAGER_SWAGGER_ENABLED=false to stop serving it.
func SwaggerEnabled() bool {
	raw := os.Getenv("MC_IAM_MANAGER_SWAGGER_ENABLED")
	if raw == "" {
		return true
	}
	enabled, err := strconv.ParseBool(raw)
	if err != nil {
		log.Printf("[WARN] invalid MC_IAM_MANAGER_SWAGGER_ENABLED=%q, using default true", raw)
		return true
	}
	return enabled
}

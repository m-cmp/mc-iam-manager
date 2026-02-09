package model

type ErrorResponse struct {
	Success bool              `json:"success"`
	Error   string            `json:"error"`
	Fields  map[string]string `json:"fields,omitempty"`
	Code    string            `json:"code,omitempty"`
}

// Error codes
const (
	ErrCodeDuplicateEmail = "DUPLICATE_EMAIL"
	ErrCodeValidation     = "VALIDATION_FAILED"
	ErrCodeServerError    = "SERVER_ERROR"
)

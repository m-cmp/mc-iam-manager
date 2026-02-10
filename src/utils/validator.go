package utils

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidateStruct validates a struct using validator tags
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

// FormatValidationError converts validation errors to user-friendly messages
func FormatValidationError(err error) string {
	if err == nil {
		return ""
	}

	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return "유효하지 않은 입력입니다"
	}

	var messages []string
	for _, e := range validationErrs {
		message := getFieldErrorMessage(e)
		messages = append(messages, message)
	}

	return strings.Join(messages, "; ")
}

// FormatValidationErrorMap returns field-specific error messages
func FormatValidationErrorMap(err error) map[string]string {
	errMap := make(map[string]string)

	if err == nil {
		return errMap
	}

	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		errMap["error"] = "유효하지 않은 입력입니다"
		return errMap
	}

	for _, e := range validationErrs {
		field := strings.ToLower(e.Field())
		errMap[field] = getFieldErrorMessage(e)
	}

	return errMap
}

func getFieldErrorMessage(e validator.FieldError) string {
	field := e.Field()

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s은(는) 필수 입력입니다", field)
	case "email":
		return "유효한 이메일 형식이 아닙니다"
	case "min":
		if e.Param() == "8" {
			return "비밀번호는 8자 이상이어야 합니다"
		}
		return fmt.Sprintf("%s은(는) 최소 %s자 이상이어야 합니다", field, e.Param())
	default:
		return fmt.Sprintf("%s 검증 실패", field)
	}
}

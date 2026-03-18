package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// defaultValidate is the shared validator instance for Struct validation.
var defaultValidate = validator.New()

// EchoValidator adapts go-playground/validator to Echo's Validator interface.
type EchoValidator struct {
	validate *validator.Validate
}

// New returns a new EchoValidator instance.
func New() *EchoValidator {
	return &EchoValidator{validate: validator.New()}
}

// Validate implements echo.Validator.
func (v *EchoValidator) Validate(i interface{}) error {
	return v.validate.Struct(i)
}

// ValidateStruct validates any struct using the default validator and its validate tags.
// Use this for gRPC handlers, services, or any non-Echo call site.
func ValidateStruct(i interface{}) error {
	if i == nil {
		return fmt.Errorf("value is required")
	}
	return defaultValidate.Struct(i)
}

// FormatValidationError converts validator.ValidationErrors into a single readable message.
// Returns the original error if it is not validator.ValidationErrors.
func FormatValidationError(err error) error {
	if err == nil {
		return nil
	}
	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}
	var parts []string
	for _, e := range validationErrs {
		parts = append(parts, fmt.Sprintf("%s: %s", e.Field(), e.Tag()))
	}
	return fmt.Errorf("validation failed: %s", strings.Join(parts, "; "))
}

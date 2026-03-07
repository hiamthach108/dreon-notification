package validator

import (
	"github.com/go-playground/validator/v10"
)

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

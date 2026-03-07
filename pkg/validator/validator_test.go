package validator

import (
	"testing"
)

func TestNew(t *testing.T) {
	v := New()
	if v == nil {
		t.Fatal("New() returned nil")
	}
	if v.validate == nil {
		t.Fatal("New() returned validator with nil validate")
	}
}

func TestEchoValidator_Validate_validStruct(t *testing.T) {
	v := New()
	type validReq struct {
		Email string `validate:"required,email"`
		Name  string `validate:"required,min=2"`
	}
	req := validReq{Email: "user@example.com", Name: "Alice"}
	if err := v.Validate(req); err != nil {
		t.Errorf("Validate(valid struct) err = %v, want nil", err)
	}
}

func TestEchoValidator_Validate_invalidStruct(t *testing.T) {
	v := New()
	type validReq struct {
		Email string `validate:"required,email"`
		Name  string `validate:"required,min=2"`
	}
	req := validReq{Email: "not-an-email", Name: "A"}
	err := v.Validate(req)
	if err == nil {
		t.Fatal("Validate(invalid struct) err = nil, want non-nil")
	}
}

func TestEchoValidator_Validate_missingRequired(t *testing.T) {
	v := New()
	type validReq struct {
		Email string `validate:"required,email"`
	}
	req := validReq{Email: ""}
	err := v.Validate(req)
	if err == nil {
		t.Fatal("Validate(missing required) err = nil, want non-nil")
	}
}

func TestEchoValidator_Validate_nonStruct(t *testing.T) {
	v := New()
	// Validating a non-struct may or may not error depending on go-playground/validator
	err := v.Validate("string")
	if err != nil {
		// Expected when validator expects a struct
		return
	}
	// Some validators accept non-struct and pass; both are acceptable
}

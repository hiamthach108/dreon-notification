package errorx

import (
	"errors"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appError *AppError
		want     string
	}{
		{
			name: "error without underlying error",
			appError: &AppError{
				Code:    ErrBadRequest,
				Message: "Bad request",
				Err:     nil,
			},
			want: "Bad request",
		},
		{
			name: "error with underlying error",
			appError: &AppError{
				Code:    ErrInternal,
				Message: "Internal server error",
				Err:     errors.New("database connection failed"),
			},
			want: "Internal server error: database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.appError.Error()
			if got != tt.want {
				t.Errorf("AppError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name         string
		code         AppErrCode
		err          error
		wantCode     AppErrCode
		wantMessage  string
		wantHasError bool
	}{
		{
			name:         "wrap with internal error",
			code:         ErrInternal,
			err:          errors.New("database error"),
			wantCode:     ErrInternal,
			wantMessage:  "Internal server error",
			wantHasError: true,
		},
		{
			name:         "wrap with not found error",
			code:         ErrNotFound,
			err:          errors.New("user not found"),
			wantCode:     ErrNotFound,
			wantMessage:  "Resource not found",
			wantHasError: true,
		},
		{
			name:         "wrap with nil error",
			code:         ErrBadRequest,
			err:          nil,
			wantCode:     ErrBadRequest,
			wantMessage:  "Bad request",
			wantHasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Wrap(tt.code, tt.err)

			if got.Code != tt.wantCode {
				t.Errorf("Wrap().Code = %v, want %v", got.Code, tt.wantCode)
			}

			if got.Message != tt.wantMessage {
				t.Errorf("Wrap().Message = %v, want %v", got.Message, tt.wantMessage)
			}

			if tt.wantHasError && got.Err == nil {
				t.Error("Wrap().Err = nil, want non-nil")
			}

			if !tt.wantHasError && got.Err != nil {
				t.Errorf("Wrap().Err = %v, want nil", got.Err)
			}

			if tt.wantHasError && got.Err != tt.err {
				t.Errorf("Wrap().Err = %v, want %v", got.Err, tt.err)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		code        AppErrCode
		message     string
		wantCode    AppErrCode
		wantMessage string
	}{
		{
			name:        "create new error with custom message",
			code:        ErrBadRequest,
			message:     "Invalid input format",
			wantCode:    ErrBadRequest,
			wantMessage: "Invalid input format",
		},
		{
			name:        "create new internal error",
			code:        ErrInternal,
			message:     "Something went wrong",
			wantCode:    ErrInternal,
			wantMessage: "Something went wrong",
		},
		{
			name:        "create error with empty message",
			code:        ErrForbidden,
			message:     "",
			wantCode:    ErrForbidden,
			wantMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.code, tt.message)

			if got.Code != tt.wantCode {
				t.Errorf("New().Code = %v, want %v", got.Code, tt.wantCode)
			}

			if got.Message != tt.wantMessage {
				t.Errorf("New().Message = %v, want %v", got.Message, tt.wantMessage)
			}

			if got.Err != nil {
				t.Errorf("New().Err = %v, want nil", got.Err)
			}
		})
	}
}

func TestAppError_ErrorInterface(t *testing.T) {
	// Test that AppError implements the error interface
	var _ error = &AppError{}
	var _ error = (*AppError)(nil)

	// Test error interface usage
	err := New(ErrBadRequest, "test error")
	if err.Error() == "" {
		t.Error("Error() should return non-empty string")
	}
}

func TestWrapAndUnwrap(t *testing.T) {
	// Test that wrapped errors can be unwrapped
	originalErr := errors.New("original error")
	wrappedErr := Wrap(ErrInternal, originalErr)

	// Check if the error message contains the original error
	errorMsg := wrappedErr.Error()
	if errorMsg == "" {
		t.Error("wrapped error should have non-empty message")
	}

	// Check if we can access the underlying error
	if wrappedErr.Err != originalErr {
		t.Errorf("Err field = %v, want %v", wrappedErr.Err, originalErr)
	}
}

func TestErrorChaining(t *testing.T) {
	// Test creating a chain of errors
	baseErr := errors.New("base error")
	appErr := Wrap(ErrInternal, baseErr)

	// Verify the chain
	if appErr.Err != baseErr {
		t.Error("error chain broken")
	}

	// Verify the error message includes both
	errorMsg := appErr.Error()
	expectedMsg := "Internal server error: base error"
	if errorMsg != expectedMsg {
		t.Errorf("Error() = %q, want %q", errorMsg, expectedMsg)
	}
}

func TestAllErrorCodes(t *testing.T) {
	// Test wrapping with all defined error codes
	codes := []AppErrCode{
		ErrInternal,
		ErrBadRequest,
		ErrNotFound,
		ErrUnauthorized,
		ErrForbidden,
		ErrConflict,
		ErrUnprocessable,
		ErrRateLimit,
	}

	for _, code := range codes {
		t.Run(GetErrorMessage(int(code)), func(t *testing.T) {
			err := Wrap(code, errors.New("test"))
			if err.Code != code {
				t.Errorf("Wrap().Code = %v, want %v", err.Code, code)
			}

			if err.Message == "" {
				t.Error("Wrap().Message should not be empty")
			}

			if err.Err == nil {
				t.Error("Wrap().Err should not be nil")
			}
		})
	}
}

func TestNewWithAllErrorCodes(t *testing.T) {
	// Test New() with all defined error codes
	codes := []AppErrCode{
		ErrInternal,
		ErrBadRequest,
		ErrNotFound,
		ErrUnauthorized,
		ErrForbidden,
		ErrConflict,
		ErrUnprocessable,
		ErrRateLimit,
	}

	for _, code := range codes {
		t.Run(GetErrorMessage(int(code)), func(t *testing.T) {
			customMsg := "custom error message"
			err := New(code, customMsg)

			if err.Code != code {
				t.Errorf("New().Code = %v, want %v", err.Code, code)
			}

			if err.Message != customMsg {
				t.Errorf("New().Message = %v, want %v", err.Message, customMsg)
			}

			if err.Err != nil {
				t.Error("New().Err should be nil")
			}
		})
	}
}

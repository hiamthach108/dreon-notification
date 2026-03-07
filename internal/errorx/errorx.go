package errorx

import "fmt"

type AppError struct {
	Code    AppErrCode
	Message string
	Err     error // optional underlying error
}

// Implement the `error` interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Wrap creates a new AppError with an existing error
func Wrap(code AppErrCode, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: GetErrorMessage(int(code)),
		Err:     err,
	}
}

// New creates a new AppError with custom message
func New(code AppErrCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// GetCode extracts the error code from an AppError
func GetCode(err error) AppErrCode {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return ErrInternal
}

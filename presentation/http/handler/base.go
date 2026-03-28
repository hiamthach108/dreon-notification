package handler

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/hiamthach108/dreon-notification/internal/errorx"
	"github.com/labstack/echo/v4"
)

type BaseResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ValidationErrItem describes one invalid field for validation error responses.
type ValidationErrItem struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrResp is the response body when validation fails.
type ValidationErrResp struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Errors  []ValidationErrItem `json:"errors"`
}

// HandleValidateBind binds the request body to a value of type T and validates it.
// Returns the bound value or the zero value and an error on bind/validation failure.
func HandleValidateBind[T any](c echo.Context) (T, error) {
	var req T
	if err := c.Bind(&req); err != nil {
		return req, err
	}
	if err := c.Validate(&req); err != nil {
		return req, err
	}
	return req, nil
}

func HandleSuccess(c echo.Context, data any) error {
	resp := BaseResp{
		Code:    http.StatusOK,
		Message: "success",
		Data:    data,
	}
	return c.JSON(http.StatusOK, resp)
}

func HandleError(c echo.Context, err error) error {
	// Validation errors: return list of invalid/missing fields
	var valErrs validator.ValidationErrors
	if errors.As(err, &valErrs) {
		errors := make([]ValidationErrItem, 0, len(valErrs))
		for _, e := range valErrs {
			errors = append(errors, ValidationErrItem{
				Field:   e.Field(),
				Message: validationTagMessage(e.Tag()),
			})
		}
		return c.JSON(http.StatusBadRequest, ValidationErrResp{
			Code:    http.StatusBadRequest,
			Message: "Validation failed",
			Errors:  errors,
		})
	}

	resp := BaseResp{}
	var appErr *errorx.AppError
	if errors.As(err, &appErr) {
		// If wrapped error is validation, still return validation response
		if errors.As(appErr.Err, &valErrs) {
			errors := make([]ValidationErrItem, 0, len(valErrs))
			for _, e := range valErrs {
				errors = append(errors, ValidationErrItem{
					Field:   e.Field(),
					Message: validationTagMessage(e.Tag()),
				})
			}
			return c.JSON(http.StatusBadRequest, ValidationErrResp{
				Code:    int(appErr.Code),
				Message: "Validation failed",
				Errors:  errors,
			})
		}
		resp.Code = int(appErr.Code)
		resp.Message = appErr.Message
		status := http.StatusInternalServerError
		if appErr.Code < 500 {
			status = int(appErr.Code)
		}
		return c.JSON(status, resp)
	}

	// fallback for unexpected errors
	resp.Code = int(errorx.ErrInternal)
	resp.Message = errorx.GetErrorMessage(int(errorx.ErrInternal))
	return c.JSON(http.StatusInternalServerError, resp)
}

// validationTagMessage returns a short message for common validator tags.
func validationTagMessage(tag string) string {
	switch tag {
	case "required":
		return "is required"
	case "min":
		return "is too short"
	case "max":
		return "is too long"
	case "email":
		return "must be a valid email"
	case "gte":
		return "must be greater or equal"
	case "lte":
		return "must be less or equal"
	case "oneof":
		return "must be one of the allowed values"
	case "omitempty":
		return "invalid"
	default:
		return "invalid"
	}
}

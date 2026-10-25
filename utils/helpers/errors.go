package helpers

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"taskmanager/utils/i18n"
)

// Code is a stable, machine-readable error code returned by the API.
type Code string

const (
	CodeBadRequest   Code = "bad_request"
	CodeNotFound     Code = "not_found"
	CodeConflict     Code = "conflict"
	CodeNoModel      Code = "no_model"
	CodeBudget       Code = "budget_exhausted"
	CodeVerification Code = "verification_failed"
	CodeInternal     Code = "internal_error"
)

// AppError is an application error: a code for clients, a translation key for
// humans, and an HTTP status for the transport layer. Args carries only
// already-localizable values; raw technical causes go to Detail, never into
// the message template.
type AppError struct {
	Code   Code
	Key    i18n.Key
	Args   map[string]any
	Detail string
	Status int
	Err    error
}

// Error renders the message in the default language. The HTTP layer renders it
// in the request language via RespondError.
func (e *AppError) Error() string { return i18n.Translate(i18n.Default(), e.Key, e.Args) }

func (e *AppError) Unwrap() error { return e.Err }

// HTTPStatus returns the status to send to a client.
func (e *AppError) HTTPStatus() int {
	if e.Status > 0 {
		return e.Status
	}
	return statusFor(e.Code)
}

// New builds an application error for a code and a message key.
func New(code Code, key i18n.Key, args map[string]any) *AppError {
	return &AppError{Code: code, Key: key, Args: args, Status: statusFor(code)}
}

// Wrap keeps the cause while exposing a code and a translated message.
func Wrap(cause error, code Code, key i18n.Key, args map[string]any) *AppError {
	err := New(code, key, args)
	err.Err = cause
	return err
}

// WrapDetail additionally exposes the raw cause on Detail. Use it when the
// technical text ( timeouts, provider errors, file paths ) must reach the
// client without being woven into the translated sentence: API error
// templates never carry {reason}/{cause} placeholders.
func WrapDetail(cause error, code Code, key i18n.Key, args map[string]any) *AppError {
	err := Wrap(cause, code, key, args)
	if cause != nil {
		err.Detail = cause.Error()
	}
	return err
}

// As extracts an application error from a chain.
func As(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// CodeOf returns the error code, or internal_error for foreign errors.
func CodeOf(err error) Code {
	if appErr, ok := As(err); ok {
		return appErr.Code
	}
	return CodeInternal
}

// StatusOf returns the HTTP status for an error.
func StatusOf(err error) int {
	if appErr, ok := As(err); ok {
		return appErr.HTTPStatus()
	}
	return http.StatusInternalServerError
}

// The error catalogue. Every error the API can produce lives here.
var (
	ErrTitleRequired    = New(CodeBadRequest, i18n.ErrTitleRequired, nil)
	ErrRoleModelNeeded  = New(CodeBadRequest, i18n.ErrRoleModelNeeded, nil)
	ErrTaskNotFound     = New(CodeNotFound, i18n.ErrTaskNotFound, nil)
	ErrInvalidJSON      = New(CodeBadRequest, i18n.ErrInvalidJSON, nil)
	ErrEmptyChain       = New(CodeNoModel, i18n.ErrEmptyChain, nil)
	ErrInvalidRoute     = New(CodeBadRequest, i18n.ErrInvalidRoute, nil)
	ErrRouteNotFound    = New(CodeNotFound, i18n.ErrRouteNotFound, nil)
	ErrContextRequired  = New(CodeBadRequest, i18n.ErrContextRequired, nil)
	ErrPlanEmpty        = New(CodeBadRequest, i18n.ErrPlanEmpty, nil)
	ErrMCPNotFound      = New(CodeNotFound, i18n.ErrMCPNotFound, nil)
	ErrMCPNameRequired  = New(CodeBadRequest, i18n.ErrMCPNameRequired, nil)
	ErrEndpointNotFound = New(CodeNotFound, i18n.ErrEndpointNotFound, nil)
	ErrBadRequest       = New(CodeBadRequest, i18n.ErrBadRequest, nil)
	ErrMethodNotAllowed = New(CodeBadRequest, i18n.ErrMethodNotAllowed, nil)
)

// ErrUnknownPipeline reports a pipeline id that is not registered.
func ErrUnknownPipeline(id string) *AppError {
	return New(CodeBadRequest, i18n.ErrUnknownPipeline, map[string]any{"pipeline": id})
}

// ErrTaskMoveInvalid reports a status a human cannot set on the board.
func ErrTaskMoveInvalid(status string) *AppError {
	return New(CodeBadRequest, i18n.ErrTaskMoveStatus, map[string]any{"status": status})
}

// ErrMCPConfig wraps a failure to read or write the MCP configuration. The
// technical cause travels on Detail; the sentence stays language-clean.
func ErrMCPConfig(cause error) *AppError {
	return WrapDetail(cause, CodeInternal, i18n.ErrMCPConfig, nil)
}

// ErrPlanGeneration wraps a planner failure while building a plan preview.
func ErrPlanGeneration(cause error) *AppError {
	return WrapDetail(cause, CodeInternal, i18n.ErrPlanFailed, nil)
}

// ErrNoModelForRole reports a role without a usable model chain.
func ErrNoModelForRole(role string) *AppError {
	return New(CodeNoModel, i18n.ErrNoModelForRole, map[string]any{"role": role})
}

// ErrBudgetExhausted reports a task that spent its token budget.
func ErrBudgetExhausted(tokens int) *AppError {
	return New(CodeBudget, i18n.ErrBudgetExhausted, map[string]any{"budget": tokens})
}

// ErrVerification wraps a verifier transport failure. The cause stays out of
// the sentence, so the verifier's own text (any language) renders only in
// detail and not inside the localized message.
func ErrVerification(cause error) *AppError {
	return WrapDetail(cause, CodeVerification, i18n.ErrVerification, nil)
}

// ErrReviewRejected reports a verdict that did not pass.
func ErrReviewRejected(score int, issues string) *AppError {
	return New(CodeVerification, i18n.MsgTaskReviewRejected, map[string]any{"score": score, "issues": issues})
}

// RespondError writes an error as {"code","key","args","message","detail"}
// with the right status. The message is rendered in the request language; the
// technical cause, when present, rides on detail untranslated.
func RespondError(c *fiber.Ctx, err error) error {
	ctx := c.UserContext()
	if appErr, ok := As(err); ok {
		return c.Status(appErr.HTTPStatus()).JSON(ErrorBody{
			Code:    string(appErr.Code),
			Key:     string(appErr.Key),
			Args:    appErr.Args,
			Message: i18n.T(ctx, appErr.Key, appErr.Args),
			Detail:  appErr.Detail,
		})
	}
	return c.Status(http.StatusInternalServerError).JSON(ErrorBody{
		Code:    string(CodeInternal),
		Key:     string(i18n.ErrInternal),
		Message: i18n.T(ctx, i18n.ErrInternal, nil),
	})
}

func statusFor(code Code) int {
	switch code {
	case CodeBadRequest:
		return http.StatusBadRequest
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeNoModel:
		return http.StatusServiceUnavailable
	case CodeBudget:
		return http.StatusConflict
	case CodeVerification:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

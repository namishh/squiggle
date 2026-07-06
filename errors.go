package main

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
)

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

type SquiggleError struct {
	Status  int
	Message string
	Code    string
}

func (e *SquiggleError) Error() string { return e.Message }

var (
	ErrRateLimited    = &SquiggleError{http.StatusTooManyRequests, "rate limit exceeded", "rate_limited"}
	ErrBanned         = &SquiggleError{http.StatusForbidden, "you are banned from posting", "banned"}
	ErrInvalidCaptcha = &SquiggleError{http.StatusForbidden, "captcha verification failed", "invalid_captcha"}
	ErrInvalidOrigin  = &SquiggleError{http.StatusForbidden, "invalid origin", "invalid_origin"}
	ErrInternal       = &SquiggleError{http.StatusInternalServerError, "internal error", "internal_error"}
	ErrEmail          = &SquiggleError{http.StatusBadRequest, "invalid email", "invalid_email"}
	ErrSite           = &SquiggleError{http.StatusBadRequest, "invalid site", "invalid_site"}
	ErrDetails        = &SquiggleError{http.StatusBadRequest, "name and message are mandatory ", "invalid_validation"}
	ErrEntryPost      = &SquiggleError{http.StatusBadRequest, "failed to save the entry ", "post_failure"}
)

func ErrValidation(msg string) error {
	return &SquiggleError{http.StatusBadRequest, msg, "validation_error"}
}

// httpErrorHandler is a method on Server so it can log through the
// server's own logger instead of a package-level global.
func (s *Server) httpErrorHandler(c *echo.Context, err error) {
	if resp, uErr := echo.UnwrapResponse(c.Response()); uErr == nil {
		if resp.Committed {
			return
		}
	}

	status := http.StatusInternalServerError
	msg := "internal error"
	code := "internal_error"

	if se, ok := errors.AsType[*SquiggleError](err); ok {
		status = se.Status
		msg = se.Message
		code = se.Code
	}

	s.logger.Error("request failed",
		"path", c.Request().URL.Path,
		"status", status,
		"code", code,
		"err", err,
	)

	c.JSON(status, ErrorResponse{Error: msg, Code: code})
}

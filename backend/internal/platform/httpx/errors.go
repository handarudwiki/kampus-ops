// Package httpx holds the HTTP envelope, error model and request helpers that
// implement the API contract conventions.
package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// AppError is a contract-shaped error: { error: { code, message, details? } }.
type AppError struct {
	Status  int
	Code    string
	Message string
	Details any
}

func (e *AppError) Error() string { return e.Code + ": " + e.Message }

func E(status int, code, msg string) *AppError {
	return &AppError{Status: status, Code: code, Message: msg}
}

func (e *AppError) With(details any) *AppError {
	c := *e
	c.Details = details
	return &c
}

func NotFound(msg string) *AppError {
	if msg == "" {
		msg = "Data tidak ditemukan"
	}
	return E(http.StatusNotFound, "NOT_FOUND", msg)
}

// Rule is a 422 business-rule violation.
func Rule(code, msg string) *AppError { return E(http.StatusUnprocessableEntity, code, msg) }

// Conflict is a 409 (schedule clash, duplicates).
func Conflict(code, msg string) *AppError { return E(http.StatusConflict, code, msg) }

func Forbidden(code, msg string) *AppError { return E(http.StatusForbidden, code, msg) }

func Unauthorized(code, msg string) *AppError { return E(http.StatusUnauthorized, code, msg) }

// Validation is a 422 VALIDATION_ERROR with field messages.
func Validation(fields map[string]string) *AppError {
	return &AppError{Status: http.StatusUnprocessableEntity, Code: "VALIDATION_ERROR", Message: "Data tidak valid", Details: fields}
}

func ValidationField(field, msg string) *AppError {
	return Validation(map[string]string{field: msg})
}

type errBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// ErrorHandler is the Fiber error handler mapping every error to the contract envelope.
func ErrorHandler(c *fiber.Ctx, err error) error {
	var ae *AppError
	var fe *fiber.Error
	var pg *pgconn.PgError
	switch {
	case errors.As(err, &ae):
	case errors.Is(err, pgx.ErrNoRows):
		ae = NotFound("")
	case errors.As(err, &fe):
		code := "HTTP_ERROR"
		switch fe.Code {
		case 404:
			code = "NOT_FOUND"
		case 405:
			code = "METHOD_NOT_ALLOWED"
		case 429:
			code = "RATE_LIMITED"
		case 413:
			code = "PAYLOAD_TOO_LARGE"
		}
		ae = E(fe.Code, code, fe.Message)
	case errors.As(err, &pg) && pg.Code == "23505":
		ae = Conflict("DUPLICATE", "Data sudah ada")
	case errors.As(err, &pg) && (pg.Code == "22P02" || pg.Code == "23503"):
		// invalid uuid text / FK to a row that is not visible -> treat as not found
		ae = NotFound("")
	default:
		slog.ErrorContext(c.UserContext(), "unhandled error", "err", err.Error(), "path", c.Path())
		ae = E(http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan pada server")
	}
	return c.Status(ae.Status).JSON(fiber.Map{"error": errBody{Code: ae.Code, Message: ae.Message, Details: ae.Details}})
}

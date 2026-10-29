package httpx

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Meta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

// OK writes { data }.
func OK(c *fiber.Ctx, data any) error {
	return c.JSON(fiber.Map{"data": data})
}

// Created writes 201 { data }.
func Created(c *fiber.Ctx, data any) error {
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": data})
}

// List writes { data, meta }.
func List(c *fiber.Ctx, data any, p Page, total int) error {
	return c.JSON(fiber.Map{"data": data, "meta": Meta{Page: p.Page, PerPage: p.PerPage, Total: total}})
}

type Page struct {
	Page    int
	PerPage int
	Q       string
}

func (p Page) Offset() int { return (p.Page - 1) * p.PerPage }

// Like returns a %q% pattern for ILIKE (empty q matches all).
func (p Page) Like() string { return "%" + strings.TrimSpace(p.Q) + "%" }

// ParsePage reads ?q=&page=&per_page= (default 20, max 100).
func ParsePage(c *fiber.Ctx) Page {
	p := Page{Page: 1, PerPage: 20, Q: strings.TrimSpace(c.Query("q"))}
	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
		p.Page = v
	}
	if v, err := strconv.Atoi(c.Query("per_page")); err == nil && v > 0 {
		p.PerPage = v
	}
	if p.PerPage > 100 {
		p.PerPage = 100
	}
	return p
}

// ParamID returns a uuid path param or a 404 when it is not a uuid.
func ParamID(c *fiber.Ctx, name string) (string, error) {
	v := c.Params(name)
	if _, err := uuid.Parse(v); err != nil {
		return "", NotFound("")
	}
	return v, nil
}

// OptUUID returns nil for empty strings, validates uuid otherwise.
func OptUUID(field, v string) (*string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	if _, err := uuid.Parse(v); err != nil {
		return nil, ValidationField(field, "harus berupa UUID")
	}
	return &v, nil
}

// Bind parses JSON body into dst; malformed JSON → VALIDATION_ERROR.
func Bind(c *fiber.Ctx, dst any) error {
	if len(c.Body()) == 0 {
		return nil
	}
	if err := c.BodyParser(dst); err != nil {
		return ValidationField("body", "JSON tidak valid: "+err.Error())
	}
	return nil
}

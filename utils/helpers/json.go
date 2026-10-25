package helpers

import "github.com/gofiber/fiber/v2"

// ErrorBody is the error payload of the API. Code is stable and machine
// readable; Key and Args let a client re-render the message in another
// language without another round trip. Detail carries the raw technical cause
// in its own language (provider errors, timeouts, paths) — never translated
// and never concatenated into the localized message.
type ErrorBody struct {
	Code    string         `json:"code"`
	Key     string         `json:"key,omitempty"`
	Args    map[string]any `json:"args,omitempty"`
	Message string         `json:"message"`
	Detail  string         `json:"detail,omitempty"`
}

func JSONOk(c *fiber.Ctx, status int, data any) error {
	return c.Status(status).JSON(data)
}

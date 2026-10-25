package middlewares

import (
	"github.com/gofiber/fiber/v2"

	"taskmanager/utils/i18n"
)

// Locale resolves the request language and stores it in the request context.
//
// Priority: ?lang / ?locale -> X-Locale -> cookie "lang" -> Accept-Language
// -> the deployment default (def). An unknown tag is skipped silently, never a
// 400: a client with a typo still gets a usable response.
func Locale(def i18n.Lang) fiber.Handler {
	return func(c *fiber.Ctx) error {
		candidates := []string{
			c.Query("lang"),
			c.Query("locale"),
			c.Get("X-Locale"),
			c.Cookies("lang"),
		}
		for _, lang := range i18n.ParseAcceptLanguage(c.Get(fiber.HeaderAcceptLanguage)) {
			candidates = append(candidates, string(lang))
		}
		candidates = append(candidates, string(def))

		lang := i18n.Resolve(candidates...)
		c.SetUserContext(i18n.WithLang(c.UserContext(), lang))
		c.Locals("lang", string(lang))
		c.Set(fiber.HeaderContentLanguage, string(lang))
		c.Vary(fiber.HeaderAcceptLanguage, "X-Locale")
		return c.Next()
	}
}

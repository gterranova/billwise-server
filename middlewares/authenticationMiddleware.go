package middlewares

import (
	"regexp"

	"it.terra9/billwise-server/util"

	"github.com/gofiber/fiber/v2"
)

func GetAuthenticatedUser(ctx *fiber.Ctx) error {
	token := ctx.Cookies(util.CookieName)
	userId, err := util.ParseToken(token)
	if err != nil {
		reBearer := regexp.MustCompile("(?i)^Bearer ")
		ts := ctx.Get("Authorization")
		if !reBearer.MatchString(ts) {
			ctx.Locals("userId", "")
			return ctx.Next()

		}
		token = ts[len("Bearer "):]
		if userId, err = util.ParseToken(token); err != nil {
			ctx.Locals("userId", "")
			return ctx.Next()
		}
	}
	ctx.Locals("userId", userId)
	return ctx.Next()
}

func IsUserAuthenticated(ctx *fiber.Ctx) error {
	userId := ctx.Locals("userId").(string)

	if len(userId) == 0 {
		return fiber.ErrUnauthorized
	}
	return ctx.Next()
}

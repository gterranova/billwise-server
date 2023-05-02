package routes

import (
	"it.terra9/billwise-server/controllers"

	"github.com/gofiber/fiber/v2"
)

func SetupPermissionRoutes(app *fiber.App) {
	route := app.Group("/api/v1")

	route.Get("/permissions", controllers.AllPermissions)
}

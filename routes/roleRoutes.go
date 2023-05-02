package routes

import (
	"it.terra9/billwise-server/controllers"

	"github.com/gofiber/fiber/v2"
)

func SetupRoleRoutes(app *fiber.App) {
	route := app.Group("/api/v1")

	route.Get("/roles", controllers.AllRoles)
	route.Get("/roles/:id", controllers.GetRole)
	route.Post("/roles", controllers.CreateRole)
	route.Put("/roles/:id", controllers.UpdateRole)
	route.Delete("/roles/:id", controllers.DeleteRole)
}

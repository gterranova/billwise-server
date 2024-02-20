package routes

import (
	"it.terra9/billwise-server/controllers"

	"github.com/gofiber/fiber/v2"
)

func SetupActivityRoutes(app *fiber.App) {
	route := app.Group("/api/v1")

	route.Get("/activities", controllers.AllActivities)
	route.Get("/activities/payable", controllers.UserPayableActivities)
	route.Get("/activities/:id", controllers.GetActivity)
	route.Post("/activities", controllers.CreateActivity)
	route.Post("/activities/import", controllers.ImportActivities)
	route.Put("/activities/:id", controllers.UpdateActivity)
	route.Delete("/activities/:id", controllers.DeleteActivity)
}

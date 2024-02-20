package routes

import (
	"it.terra9/billwise-server/controllers"

	"github.com/gofiber/fiber/v2"
)

func SetupTaskRoutes(app *fiber.App) {
	route := app.Group("/api/v1")

	route.Get("/tasks", controllers.AllTasks)
	route.Get("/tasks/:id", controllers.GetTask)
	route.Get("/tasks/:id/activities", controllers.GetTaskActivities)
	route.Post("/tasks", controllers.CreateTask)
	route.Post("/tasks/import", controllers.ImportTasks)
	route.Put("/tasks/:id", controllers.UpdateTask)
	route.Delete("/tasks/:id", controllers.DeleteTask)
}

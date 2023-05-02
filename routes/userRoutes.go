package routes

import (
	"it.terra9/billwise-server/controllers"

	"github.com/gofiber/fiber/v2"
)

func SetupUserRoutes(app *fiber.App) {
	route := app.Group("/api/v1")

	route.Get("/users", controllers.AllUsers)
	route.Get("/users/:id", controllers.GetUser)
	//route.Get("/users/:id/activities", controllers.GetUserActivities)
	//route.Get("/users/:id/tasks", controllers.GetUserTasks)
	route.Post("/users", controllers.CreateUser)
	route.Put("/users/:id", controllers.UpdateUser)
	route.Delete("/users/:id", controllers.DeleteUser)
}

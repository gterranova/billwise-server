package routes

import (
	"it.terra9/billwise-server/controllers"

	"github.com/gofiber/fiber/v2"
)

func SetupPublicAuthRoutes(app *fiber.App) {
	route := app.Group("/api/v1")

	route.Post("/register", controllers.Register)
	route.Post("/login", controllers.Login)
}

func SetupAuthRoutes(app *fiber.App) {
	route := app.Group("/api/v1")

	route.Post("/logout", controllers.Logout)
	route.Get("/me", controllers.GetCurrentUser)
	route.Put("/me", controllers.UpdateCurrentUserInfo)
	route.Put("/me/password", controllers.UpdateCurrentUserPassword)
	route.Post("/me/image", controllers.UpdateCurrentUserProfileImage)
}

package routes

import (
	"github.com/gofiber/fiber/v2"
	"it.terra9/billwise-server/middlewares"
)

func Setup(app *fiber.App) {

	app.Use(middlewares.GetAuthenticatedUser)

	SetupPublicAuthRoutes(app)
	//SetupSwaggerRoute(app)

	app.Use(middlewares.IsUserAuthenticated) // All the routes defined below this call require the user to be authenticated

	SetupAuthRoutes(app)
	SetupUserRoutes(app)
	SetupRoleRoutes(app)
	SetupPermissionRoutes(app)

	SetupTaskRoutes(app)
	SetupActivityRoutes(app)
	SetupAccountingRoutes(app)
	SetupInvoiceRoutes(app)

	// Serve static folders so we can access the uploaded images
	route := app.Group("/api/v1")

	route.Static("/uploads", "./uploads") // The first param is the URL and the second is the folder where is stored
}

package routes

import (
	"it.terra9/billwise-server/controllers"

	"github.com/gofiber/fiber/v2"
)

func SetupAccountingRoutes(app *fiber.App) {
	route := app.Group("/api/v1")

	route.Get("/accounting", controllers.AllAccountingDocuments)
	route.Get("/accounting/:id", controllers.GetAccountingDocument)
	route.Post("/accounting", controllers.CreateAccountingDocument)
	route.Post("/accounting/import", controllers.ImportAccountingDocument)
	route.Put("/accounting/:id", controllers.UpdateAccountingDocument)
	route.Delete("/accounting/:id", controllers.DeleteAccountingDocument)
}

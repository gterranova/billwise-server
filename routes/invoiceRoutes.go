package routes

import (
	"it.terra9/billwise-server/controllers"

	"github.com/gofiber/fiber/v2"
)

func SetupInvoiceRoutes(app *fiber.App) {
	route := app.Group("/api/v1")

	route.Get("/invoices", controllers.AllInvoices)
	route.Get("/invoices/:id", controllers.GetInvoice)
	route.Post("/invoices", controllers.CreateInvoice)
	route.Put("/invoices/:id", controllers.UpdateInvoice)
	route.Delete("/invoices/:id", controllers.DeleteInvoice)
}

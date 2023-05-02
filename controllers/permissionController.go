package controllers

import (
	"it.terra9/billwise-server/database"
	"it.terra9/billwise-server/middlewares"
	"it.terra9/billwise-server/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func AllPermissions(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "permissions"); err != nil {
		return err
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	return ctx.JSON(
		models.Paginate(database.DB, &models.Permission{}, page),
	)
}

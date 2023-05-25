package controllers

import (
	"strconv"

	"it.terra9/billwise-server/database"
	"it.terra9/billwise-server/middlewares"
	"it.terra9/billwise-server/models"
	"it.terra9/billwise-server/util"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func AllUsers(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "users"); err != nil {
		return err
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	return ctx.JSON(
		models.Paginate(database.DB, &models.User{}, page),
	)
}

func GetUser(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "users"); err != nil {
		return err
	}
	userId, err := util.ParseID(ctx.Params("id"))
	if err != nil {
		return err
	}
	var user models.User
	err = database.DB.Preload("Role").First(&user, "id = ?", userId).Error
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound)
	}
	return ctx.JSON(user)
}

func CreateUser(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "users"); err != nil {
		return err
	}
	var user models.User
	if err := ctx.BodyParser(&user); err != nil {
		return err
	}
	user.SetPassword(util.Config.DefaultUserPassword)
	result := database.DB.Create(&user)
	if result.Error != nil {
		return result.Error
	}
	return ctx.JSON(user)
}

func UpdateUser(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "users"); err != nil {
		return err
	}
	userId, err := util.ParseID(ctx.Params("id"))
	if err != nil {
		return err
	}
	var user models.User
	if err := ctx.BodyParser(&user); err != nil {
		return err
	}
	user.ID = userId
	user.RoleId = user.Role.ID
	result := database.DB.Model(&user).Updates(user)
	if result.Error != nil {
		return err
	}
	return ctx.JSON(user)
}

func DeleteUser(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "users"); err != nil {
		return err
	}
	userId, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return err
	}

	user := models.User{
		ID: userId,
	}

	database.DB.Model(&user).Association("Activities").Clear() // Remove existing many2many before deleting the role

	result := database.DB.Delete(&user)
	if result.Error != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound)
	}
	ctx.Status(fiber.StatusNoContent)
	return nil
}

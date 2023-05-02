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

func AllRoles(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "roles"); err != nil {
		return err
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	return ctx.JSON(
		models.Paginate(database.DB, &models.Role{}, page),
	)
}

func GetRole(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "roles"); err != nil {
		return err
	}
	roleId, err := uuid.Parse(ctx.Params("id"))

	if err != nil {
		return err
	}
	var role models.Role
	err = database.DB.Preload("Permissions").First(&role, "id = ?", roleId).Error
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound)
	}
	return ctx.JSON(role)
}

type RoleDto struct {
	Name        string      `json:"name"`
	Permissions []uuid.UUID `json:"permissions"`
}

func CreateRole(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "roles"); err != nil {
		return err
	}
	var roleDto RoleDto
	if err := ctx.BodyParser(&roleDto); err != nil {
		return err
	}

	var permissions = make([]models.Permission, len(roleDto.Permissions))
	for index, permissionId := range roleDto.Permissions {
		permissions[index] = models.Permission{
			ID: permissionId,
		}
	}

	var role = models.Role{
		Name:        roleDto.Name,
		Permissions: permissions,
	}

	result := database.DB.Create(&role)
	if result.Error != nil {
		return result.Error
	}
	return ctx.JSON(role)
}

func UpdateRole(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "roles"); err != nil {
		return err
	}
	roleId, err := util.ParseID(ctx.Params("id"))
	if err != nil {
		return err
	}
	var roleDto RoleDto
	if err := ctx.BodyParser(&roleDto); err != nil {
		return err
	}
	var permissions = make([]models.Permission, len(roleDto.Permissions))
	for index, permissionId := range roleDto.Permissions {
		permissions[index] = models.Permission{
			ID: permissionId,
		}
	}
	var role = models.Role{
		ID:   roleId,
		Name: roleDto.Name,
	}
	database.DB.Model(&role).Association("Permissions").Replace(&permissions) // Replace existing role_permissions many2many
	result := database.DB.Model(&role).Updates(role)
	if result.Error != nil {
		return err
	}
	return ctx.JSON(role)
}

func DeleteRole(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "roles"); err != nil {
		return err
	}
	roleId, err := util.ParseID(ctx.Params("id"))
	if err != nil {
		return err
	}
	role := models.Role{
		ID: roleId,
	}
	database.DB.Model(&role).Association("Permissions").Clear() // Remove existing role_permissions many2many before deleting the role
	database.DB.Delete(&role)
	ctx.Status(fiber.StatusNoContent)
	return nil
}

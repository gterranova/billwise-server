package middlewares

import (
	"it.terra9/billwise-server/database"
	"it.terra9/billwise-server/models"
	"it.terra9/billwise-server/util"

	"github.com/gofiber/fiber/v2"
)

// Naive way of checking if the logged in user has permissions to access the requested resource
// TODO: Improve the authorization middleware
func IsUserAuthorized(ctx *fiber.Ctx, resource string) error {
	id := ctx.Locals("userId").(string)

	if len(id) == 0 {
		return fiber.ErrUnauthorized
	}

	userId, _ := util.ParseID(id)
	user := models.User{
		ID: userId,
	}

	database.DB.Preload("Role").Find(&user)

	role := models.Role{
		ID: user.RoleId,
	}

	database.DB.Preload("Permissions").Find(&role)

	if ctx.Method() == "GET" {
		// A user with view or edit permissions over the resource can permorm a GET request
		for _, permission := range role.Permissions {
			if permission.Name == "view_"+resource || permission.Name == "edit_"+resource {
				return nil
			}
		}
	} else {
		// A user only with edit permissions over the resource can permorm a POST/PUT/DELETE request
		for _, permission := range role.Permissions {
			if permission.Name == "edit_"+resource {
				return nil
			}
		}
	}

	return fiber.ErrUnauthorized
}

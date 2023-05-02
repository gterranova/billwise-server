package controllers

import (
	"path/filepath"
	"time"

	"it.terra9/billwise-server/database"
	"it.terra9/billwise-server/models"
	"it.terra9/billwise-server/util"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// This method allows a user to self-register.
func Register(ctx *fiber.Ctx) error {
	registerDto := struct {
		ID              uuid.UUID `json:"id"`
		FirstName       string    `json:"firstName"`
		LastName        string    `json:"lastName"`
		Email           string    `json:"email"`
		Password        string    `json:"password"`
		PasswordConfirm string    `json:"passwordConfirm"`
		RoleId          uuid.UUID `json:"roleId"`
	}{}

	if err := ctx.BodyParser(&registerDto); err != nil {
		return err
	}

	if registerDto.Password != registerDto.PasswordConfirm {
		ctx.Status(fiber.ErrBadRequest.Code)
		return ctx.JSON(fiber.Map{
			"error": "passwords do not match",
		})
	}

	user := models.User{
		FirstName: registerDto.FirstName,
		LastName:  registerDto.LastName,
		Email:     registerDto.Email,
		RoleId:    registerDto.RoleId,
	}
	user.SetPassword(registerDto.Password)

	err := database.DB.Create(&user).Error
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "successfully registered",
	})
}

func Login(ctx *fiber.Ctx) error {
	loginDto := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{}

	if err := ctx.BodyParser(&loginDto); err != nil {
		return err
	}

	user := models.User{
		Email: loginDto.Email,
	}

	result := database.DB.Where(&user).First(&user)
	if result.Error != nil || result.RowsAffected == 0 {
		return fiber.ErrUnauthorized
	}

	if err := user.VerifyPassword(loginDto.Password); err != nil {
		return fiber.ErrUnauthorized
	}

	expirationTime := time.Now().Add(24 * time.Hour)

	token, err := util.CreateToken(user.ID.String(), expirationTime)

	if err != nil {
		return err
	}

	ctx.Cookie(&fiber.Cookie{
		Name:     util.CookieName,
		Value:    token,
		Expires:  expirationTime,
		HTTPOnly: false,
		SameSite: "lax",
	})

	ctx.Set("Authorization", "Bearer "+token)

	return ctx.JSON(fiber.Map{
		"message": "successfully logged in",
	})
}

func Logout(ctx *fiber.Ctx) error {
	ctx.Cookie(&fiber.Cookie{
		Name:     util.CookieName,
		Expires:  time.Now().Add(-(2 * time.Hour)), // Set expiry date to the past
		HTTPOnly: true,
		SameSite: "lax",
	})
	ctx.Locals("userId", "")
	return ctx.JSON(fiber.Map{
		"message": "successfully logged out",
	})
}

func GetLoggedUser(ctx *fiber.Ctx, user *models.User) error {
	userId := ctx.Locals("userId")
	return database.DB.Preload("Role").First(&user, "id = ?", userId).Error
}

func GetCurrentUser(ctx *fiber.Ctx) error {
	var user models.User
	err := GetLoggedUser(ctx, &user)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound)
	}
	return ctx.JSON(user)
}

func UpdateCurrentUserInfo(ctx *fiber.Ctx) error {
	userId := ctx.Locals("userId").(string)
	var user models.User
	if err := ctx.BodyParser(&user); err != nil {
		return err
	}
	id, _ := util.ParseID(userId)
	user.ID = id
	result := database.DB.Model(&user).Updates(user)
	if result.Error != nil {
		return result.Error
	}
	return ctx.JSON(user)
}

func UpdateCurrentUserPassword(ctx *fiber.Ctx) error {

	userId := ctx.Locals("userId").(string)
	updatePasswordDto := struct {
		Password        string `json:"password"`
		PasswordConfirm string `json:"passwordConfirm"`
	}{}

	if err := ctx.BodyParser(&updatePasswordDto); err != nil {
		return err
	}

	if updatePasswordDto.Password != updatePasswordDto.PasswordConfirm {
		ctx.Status(fiber.ErrBadRequest.Code)
		return ctx.JSON(fiber.Map{
			"error": "passwords do to match",
		})
	}

	// To decrypt the original StringToEncrypt
	decText, err := util.Decrypt(updatePasswordDto.Password, util.Config.ApiSecret)
	if err != nil {
		return ctx.JSON(fiber.Map{
			"error": "error decrypting your encrypted text",
		})
	}

	id, _ := util.ParseID(userId)
	user := models.User{
		ID: id,
	}
	user.SetPassword(decText)
	result := database.DB.Model(&user).Updates(user)
	if result.Error != nil {
		return result.Error
	}
	return ctx.JSON(fiber.Map{
		"message": "password successfully changed",
	})
}

func UpdateCurrentUserProfileImage(ctx *fiber.Ctx) error {
	userId := ctx.Locals("userId").(string)
	form, err := ctx.MultipartForm()
	if err != nil {
		return err
	}

	files := form.File["image"] // This is the key of the file when we send it in the form-data request body
	filename := uuid.New().String()

	// File format = guid.extension
	for _, file := range files {
		filename = filename + filepath.Ext(file.Filename)
		if err := ctx.SaveFile(file, "./uploads/"+filename); err != nil {
			return err
		}
	}

	id, _ := util.ParseID(userId)
	user := models.User{
		ID:       id,
		ImageUrl: ctx.BaseURL() + "/api/uploads/" + filename,
	}
	result := database.DB.Model(&user).Updates(user)
	if result.Error != nil {
		return err
	}

	return ctx.JSON(fiber.Map{
		"url": user.ImageUrl,
	})
}

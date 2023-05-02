package controllers

import (
	"path/filepath"
	"strconv"

	cmd "it.terra9/billwise-server/cmd/importCommands"
	"it.terra9/billwise-server/database"
	"it.terra9/billwise-server/middlewares"
	"it.terra9/billwise-server/models"
	"it.terra9/billwise-server/util"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func AllActivities(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "activities"); err != nil {
		return err
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	//userId, _ := models.ParseID(ctx.Locals("userId").(string))

	tx := database.DB.Model(&models.Activity{}).
		Preload("User")

	return ctx.JSON(
		models.Paginate(tx, &models.Activity{}, page),
	)
}

func GetActivity(ctx *fiber.Ctx) (err error) {
	if err = middlewares.IsUserAuthorized(ctx, "activities"); err != nil {
		return
	}
	activityId, err := util.ParseID(ctx.Params("id"))
	if err != nil {
		return
	}
	activity := models.GetActivityById(database.DB.Set("userId", ctx.Locals("userId")), activityId)
	if activity == nil {
		return fiber.NewError(fiber.StatusNotFound)
	}
	return ctx.JSON(activity)
}

func CreateActivity(ctx *fiber.Ctx) (err error) {
	if err := middlewares.IsUserAuthorized(ctx, "activities"); err != nil {
		return err
	}
	var activity models.Activity
	if err = ctx.BodyParser(&activity); err != nil {
		return err
	}

	if err = activity.Create(database.DB.Set("userId", ctx.Locals("userId"))); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return ctx.JSON(activity)
}

func UpdateActivity(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "activities"); err != nil {
		return err
	}
	activityId, err := util.ParseID(ctx.Params("id"))
	if err != nil {
		return err
	}
	var activity models.Activity
	if err := ctx.BodyParser(&activity); err != nil {
		return err
	}
	activity.ID = activityId

	if err = activity.Update(database.DB.Set("userId", ctx.Locals("userId"))); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return ctx.JSON(activity)
}

func DeleteActivity(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "activities"); err != nil {
		return err
	}
	activityId, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return err
	}

	var oldActivity models.Activity
	if err := database.DB.Set("userId", ctx.Locals("userId")).Preload("Task").First(&oldActivity, "id = ?", activityId).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound)
	}

	if result := database.DB.Set("userId", ctx.Locals("userId")).Set("oldActivity", &oldActivity).Delete(&models.Activity{}, activityId); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound)
	}

	ctx.Status(fiber.StatusNoContent)
	return nil
}

func UserPayableActivities(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "activities"); err != nil {
		return err
	}
	userId, _ := util.ParseID(ctx.Locals("userId").(string))
	var activities []models.Activity
	if err := database.DB.Set("userId", ctx.Locals("userId")).
		Where("invoice_id is null and activities.status = ? and activities.user_id = ?", models.PaymentMade, userId).
		Preload("ReferenceDocument").
		Preload("Task").Order("activities.date desc").Find(&activities).Error; err != nil {
		// do nothing
		return ctx.JSON(activities)
	}
	return ctx.JSON(activities)
}

func ImportActivities(ctx *fiber.Ctx) error {
	form, err := ctx.MultipartForm()
	if err != nil {
		return err
	}

	files := form.File["file"] // This is the key of the file when we send it in the form-data request body
	filename := uuid.New().String()

	// File format = guid.extension
	for _, file := range files {
		filename = filename + filepath.Ext(file.Filename)
		if err := ctx.SaveFile(file, "./uploads/"+filename); err != nil {
			return err
		}
	}

	if err = cmd.ImportActivitiesFromFile(database.DB.Set("userId", ctx.Locals("userId")), "./uploads/"+filename); err != nil {
		return fiber.NewError(fiber.StatusNotAcceptable)
	}

	return ctx.JSON(fiber.Map{
		"url": "./uploads/" + filename,
	})
}

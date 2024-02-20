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

func AllTasks(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "tasks"); err != nil {
		return err
	}
	var task models.Task
	userId, _ := util.ParseID(ctx.Locals("userId").(string))

	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	return ctx.JSON(
		models.Paginate(task.WithUserStats(database.DB, userId).Order("tasks.name asc"), &models.Task{}, page),
	)
}

func GetTask(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "tasks"); err != nil {
		return err
	}
	taskId, err := util.ParseID(ctx.Params("id"))
	if err != nil {
		return err
	}
	userId, _ := util.ParseID(ctx.Locals("userId").(string))

	task := models.GetTaskById(database.DB.Set("userId", ctx.Locals("userId")), taskId, userId)
	if task == nil {
		return fiber.NewError(fiber.StatusNotFound)
	}

	return ctx.JSON(task)
}

func GetTaskActivities(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "tasks"); err != nil {
		return err
	}
	taskId, err := util.ParseID(ctx.Params("id"))
	if err != nil {
		return err
	}
	//userId, _ := models.ParseID(ctx.Locals("userId").(string))

	var activities []models.Activity
	cond := &models.Activity{TaskID: taskId}
	err = database.DB.
		Preload("User").Order("activities.date desc").Find(&activities, cond).Error
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound)
	}
	return ctx.JSON(activities)
}

func CreateTask(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "tasks"); err != nil {
		return err
	}
	var task models.Task
	if err := ctx.BodyParser(&task); err != nil {
		return err
	}
	result := database.DB.Set("userId", ctx.Locals("userId")).Create(&task)
	if result.Error != nil {
		return result.Error
	}

	return ctx.JSON(task)
}

func UpdateTask(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "tasks"); err != nil {
		return err
	}
	taskId, err := util.ParseID(ctx.Params("id"))
	if err != nil {
		return err
	}
	task := models.GetTaskById(database.DB, taskId, taskId)
	if err := ctx.BodyParser(&task); err != nil {
		return err
	}

	if err := task.Update(database.DB.Set("userId", ctx.Locals("userId"))); err != nil {
		return err
	}

	return ctx.JSON(task)
}

func DeleteTask(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "tasks"); err != nil {
		return err
	}
	taskId, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return err
	}
	var oldTask models.Task
	if err := database.DB.First(&oldTask, "id = ?", taskId).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound)
	}
	if result := database.DB.Set("userId", ctx.Locals("userId")).
		Set("oldTask", &oldTask).
		Delete(&models.Task{}, taskId); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound)
	}

	ctx.Status(fiber.StatusNoContent)
	return nil
}

func ImportTasks(ctx *fiber.Ctx) error {
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

	if err = cmd.ImportTasksFromFile(database.DB.Set("userId", ctx.Locals("userId")), "./uploads/"+filename); err != nil {
		return fiber.NewError(fiber.StatusNotAcceptable)
	}

	return ctx.JSON(fiber.Map{
		"url": "./uploads/" + filename,
	})
}

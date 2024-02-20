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

func AllAccountingDocuments(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "activities"); err != nil {
		return err
	}
	var accountingDocument models.AccountingDocument
	userId, _ := util.ParseID(ctx.Locals("userId").(string))

	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	return ctx.JSON(
		models.Paginate(accountingDocument.WithUserStats(database.DB, userId), &models.AccountingDocument{}, page),
	)
}

func GetAccountingDocument(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "activities"); err != nil {
		return err
	}
	accountingId, err := util.ParseID(ctx.Params("id"))
	if err != nil {
		return err
	}
	userId, _ := util.ParseID(ctx.Locals("userId").(string))

	accountingDocument := models.GetAccountingDocumentById(
		database.DB.Set("userId", ctx.Locals("userId")),
		accountingId, userId,
	)
	if accountingDocument == nil {
		return fiber.NewError(fiber.StatusNotFound)
	}
	return ctx.JSON(accountingDocument)
}

func CreateAccountingDocument(ctx *fiber.Ctx) (err error) {
	if err = middlewares.IsUserAuthorized(ctx, "activities"); err != nil {
		return
	}
	var accountingDocument models.AccountingDocument
	if err = ctx.BodyParser(&accountingDocument); err != nil {
		return
	}
	//userId, _ := models.ParseID(ctx.Locals("userId").(string))
	for i := range accountingDocument.Activities {
		if act := models.GetActivityById(database.DB, accountingDocument.Activities[i].ID); act != nil {
			accountingDocument.Activities[i] = act
		}
	}
	if result := database.DB.Set("userId", ctx.Locals("userId")).
		Omit("Activities.*").Create(&accountingDocument); result.Error != nil {
		return result.Error
	}

	return ctx.JSON(accountingDocument)
}

func UpdateAccountingDocument(ctx *fiber.Ctx) (err error) {
	if err = middlewares.IsUserAuthorized(ctx, "activities"); err != nil {
		return err
	}
	accountingId, err := util.ParseID(ctx.Params("id"))
	if err != nil {
		return err
	}

	userId, _ := util.ParseID(ctx.Locals("userId").(string))
	var accountingDocument models.AccountingDocument
	if err = ctx.BodyParser(&accountingDocument); err != nil {
		return err
	}
	accountingDocument.ID = accountingId
	for i := range accountingDocument.Activities {
		if act := models.GetActivityById(database.DB, accountingDocument.Activities[i].ID); act != nil {
			accountingDocument.Activities[i] = act
		}
	}
	//models.ReplaceAccountingDocumentActivities(database.DB.
	//	Set("userId", ctx.Locals("userId")),
	//	accountingId,
	//	accountingDocument.Activities)

	if err = accountingDocument.Update(
		database.DB.
			Set("userId", ctx.Locals("userId"))); err != nil {
		return err

	}

	//if err := accountingDocument.UpdateUserStats(database.DB); err != nil {
	//	return err
	//}

	//if err = database.DB.Set("userId", ctx.Locals("userId")).
	//	Model(accountingDocument).Association("Activities").Replace(&accountingDocument.Activities); err != nil {

	//}

	newAccountingDocument := models.GetAccountingDocumentById(database.DB, accountingDocument.ID, userId)
	if newAccountingDocument == nil {
		return fiber.NewError(fiber.StatusNotFound)
	}
	return ctx.JSON(newAccountingDocument)
}

func DeleteAccountingDocument(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "activities"); err != nil {
		return err
	}
	accountingId, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return err
	}

	if err = models.ReplaceAccountingDocumentActivities(database.DB.Set("userId", ctx.Locals("userId")),
		accountingId,
		[]*models.Activity{}); err != nil {
		return err
	}
	if err = models.ReplaceAccountingUserStats(database.DB.Set("userId", ctx.Locals("userId")),
		accountingId,
		[]*models.AccountingDocumentUserStats{}); err != nil {
		return err
	}

	var oldAccountingDocument models.AccountingDocument
	if err := database.DB.Preload("Task").First(&oldAccountingDocument, "id = ?", accountingId).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound)
	}

	if result := database.DB.Set("userId", ctx.Locals("userId")).Set("oldAccountingDocument", &oldAccountingDocument).Delete(&models.AccountingDocument{}, accountingId); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound)
	}
	/*
		if err := oldAccountingDocument.DeleteUserStats(database.DB); err != nil {
			return err
		}

		// update associated task
		task := models.Task{ID: oldAccountingDocument.TaskID}
		if err = task.UpdateUserStats(database.DB.Set("userId", ctx.Locals("userId"))); err != nil {
			return err
		}
	*/
	ctx.Status(fiber.StatusNoContent)
	return nil
}

func ImportAccountingDocument(ctx *fiber.Ctx) error {
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

	if err = cmd.ImportAccountingDocumentFromFile(database.DB, "./uploads/"+filename); err != nil {
		return fiber.NewError(fiber.StatusNotAcceptable, err.Error())
	}

	return ctx.JSON(fiber.Map{
		"url": "./uploads/" + filename,
	})
}

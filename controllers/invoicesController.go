package controllers

import (
	"strconv"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"it.terra9/billwise-server/database"
	"it.terra9/billwise-server/middlewares"
	"it.terra9/billwise-server/models"
	"it.terra9/billwise-server/util"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Invoice struct {
	ID             uuid.UUID         `gorm:"type:uuid;primaryKey;default:(uuid_generate_v4())" json:"id,omitempty"`
	UserID         uuid.UUID         `gorm:"not null" json:"user_id"`
	User           models.User       `gorm:"foreignKey:UserID"`
	Date           datatypes.Date    `json:"date"`
	DocumentNumber int               `json:"document_number"`
	DocumentAmount *float64          `json:"document_amount"`
	Activities     []models.Activity `gorm:"constraint:OnDelete:SET NULL;" json:"activities"`
}

func (d *Invoice) Count(db *gorm.DB) int64 {
	var totalInvoices int64
	db.Model(&Invoice{}).Count(&totalInvoices)
	return totalInvoices
}

func (d *Invoice) Take(db *gorm.DB, limit int, offset int) interface{} {
	var documents []Invoice
	if limit > 0 {
		db = db.Offset(offset).Limit(limit)
	}
	db.Preload("Activities").Find(&documents)
	return documents
}

func AllInvoices(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "activities"); err != nil {
		return err
	}
	userId, _ := util.ParseID(ctx.Locals("userId").(string))

	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	return ctx.JSON(
		models.Paginate((&models.Invoice{}).ForUser(database.DB, userId), &Invoice{}, page),
	)
}

func GetInvoice(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "activities"); err != nil {
		return err
	}
	invoiceId, err := util.ParseID(ctx.Params("id"))
	if err != nil {
		return err
	}
	userId, _ := util.ParseID(ctx.Locals("userId").(string))

	invoice := models.GetInvoiceById(database.DB.Set("userId", ctx.Locals("userId")), invoiceId, userId)
	if invoice == nil {
		return fiber.NewError(fiber.StatusNotFound)
	}

	if err = database.DB.Select("activities.*, 1.0 * activities.seconds_billed/accounting_document_user_stats.total_user_seconds * accounting_document_user_stats.total_user_amounts as payable_user_base").
		Joins("left join accounting_document_user_stats on accounting_document_user_stats.accounting_document_id = activities.reference_document_id and accounting_document_user_stats.user_id = ?", userId).
		Where("invoice_id = ? and activities.user_id = ?", invoice.ID, userId).
		Preload("ReferenceDocument").
		Preload("Task").Order("activities.date desc").Find(&invoice.Activities).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound)
	}
	return ctx.JSON(invoice)
}

func CreateInvoice(ctx *fiber.Ctx) (err error) {
	if err := middlewares.IsUserAuthorized(ctx, "activities"); err != nil {
		return err
	}
	var invoice models.Invoice
	invoice.UserID, _ = util.ParseID(ctx.Locals("userId").(string))

	if err := ctx.BodyParser(&invoice); err != nil {
		return err
	}
	for i := range invoice.Activities {
		if act := models.GetActivityById(database.DB, invoice.Activities[i].ID); act != nil {
			invoice.Activities[i] = act
		}
	}
	if err = invoice.Create(database.DB.Set("userId", ctx.Locals("userId"))); err != nil {
		return
	}

	return ctx.JSON(invoice)
}

func UpdateInvoice(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "activities"); err != nil {
		return err
	}
	invoiceId, err := util.ParseID(ctx.Params("id"))
	if err != nil {
		return err
	}

	var invoice models.Invoice
	oldActivities := make([]*models.Activity, 0)
	if result := database.DB.Preload("Activities").Find(&invoice, invoiceId); result.Error != nil {
		return result.Error
	} else {
		oldActivities = append(oldActivities, invoice.Activities...)
	}
	if err := ctx.BodyParser(&invoice); err != nil {
		return err
	}
	invoice.ID = invoiceId
	invoice.UserID, _ = util.ParseID(ctx.Locals("userId").(string))
	for i := range invoice.Activities {
		if act := models.GetActivityById(database.DB, invoice.Activities[i].ID); act != nil {
			invoice.Activities[i] = act
		}
	}
	if err = invoice.Update(
		database.DB.
			Set("userId", ctx.Locals("userId")).Set("oldActivities", oldActivities)); err != nil {
		return err

	}

	return ctx.JSON(invoice)
}

func DeleteInvoice(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "activities"); err != nil {
		return err
	}
	invoiceId, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return err
	}

	var invoice models.Invoice
	oldActivities := make([]*models.Activity, 0)
	if result := database.DB.Preload("Activities").Find(&invoice, invoiceId); result.Error != nil {
		return result.Error
	} else {
		oldActivities = append(oldActivities, invoice.Activities...)
	}

	if err := models.ReplaceInvoiceActivities(database.DB, invoiceId, []*models.Activity{}); err != nil {
		panic(err)
	}

	if err := database.DB.Set("userId", ctx.Locals("userId")).Set("oldActivities", oldActivities).Delete(&models.Invoice{}, invoiceId).Error; err != nil {
		return err
	}

	ctx.Status(fiber.StatusNoContent)
	return nil
}

/*
func updateInvoicesReferences() error {

	if tx := database.DB.Exec(`
	UPDATE activities SET invoice_id = NULL WHERE activities.invoice_id NOT IN (
		SELECT invoices.id FROM activities LEFT JOIN invoices on activities.invoice_id = invoices.id WHERE invoices.id is not null
	)`); tx.Error != nil {
		return tx.Error
	}
	return nil

}
*/

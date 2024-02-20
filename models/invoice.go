package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Invoice struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:(uuid_generate_v4())" json:"id,omitempty"`
	UserID         uuid.UUID      `gorm:"not null" json:"user_id"`
	User           User           `gorm:"foreignKey:UserID"`
	Date           datatypes.Date `json:"date"`
	DocumentNumber int            `json:"document_number"`
	DocumentAmount *float64       `json:"document_amount"`
	Activities     []*Activity    `gorm:"constraint:OnDelete:SET NULL;" json:"activities"`
}

func (d *Invoice) Count(db *gorm.DB) int64 {
	var totalDocuments int64
	db.Model(&Invoice{}).Count(&totalDocuments)
	return totalDocuments
}

func (d *Invoice) Take(db *gorm.DB, limit int, offset int) interface{} {
	var documents []Invoice
	if limit > 0 {
		db = db.Offset(offset).Limit(limit)
	}
	db.Preload("Task").Preload("Activities").Order("invoices.date desc, invoices.document_number desc").Find(&documents)
	return documents
}

func GetInvoiceById(db *gorm.DB, id uuid.UUID, userId uuid.UUID) *Invoice {
	invoice := Invoice{}
	if err := invoice.ForUser(db.Preload("User").Preload("Activities"), userId).
		Where("invoices.id = ?", id).First(&invoice).Error; err != nil {
		return nil
	}
	return &invoice
}

func (d *Invoice) ForUser(db *gorm.DB, userId uuid.UUID) *gorm.DB {
	return db.Where("invoices.user_id = ?", userId)
}

func (d *Invoice) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.Select("Date", "UserID", "DocumentNumber", "DocumentAmount")
	return nil
}

func (d *Invoice) Create(db *gorm.DB) (err error) {
	return db.Omit("Activities.*").Create(d).Error
}

/*
	func (d *Invoice) AfterCreate(tx *gorm.DB) error {
		return d.AfterSave(tx.Session(&gorm.Session{NewDB: true}).Set("userId", util.SessionUserID(tx).String()))
	}
*/

func (d *Invoice) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.Select("Date", "UserID", "DocumentNumber", "DocumentAmount")
	return nil
}

func (d *Invoice) Update(tx *gorm.DB) (err error) {

	return tx.Omit("Activities").Preload("User").Model(d).
		Updates(d).Error
}

func ReplaceInvoiceActivities(db *gorm.DB, invoiceId uuid.UUID, activities []*Activity) (err error) {

	if err = db.Exec(`UPDATE activities SET invoice_id = NULL WHERE invoice_id = ?`, invoiceId).Error; err != nil {
		return err
	}
	if len(activities) > 0 {
		activityIds := make([]uuid.UUID, len(activities))
		for i := range activities {
			activityIds[i] = activities[i].ID
		}
		return db.Exec(`UPDATE activities SET invoice_id=? WHERE activities.id IN ?`,
			invoiceId, activityIds).Error
	}
	return nil
}

func (d *Invoice) AfterSave(tx *gorm.DB) (err error) {

	taskToUpdate := make(map[uuid.UUID]uint)
	tasksList := make([]uuid.UUID, 0)
	var invoice Invoice
	oldActivities := make([]*Activity, 0)
	if result := tx.Preload("Activities").Find(&invoice, d.ID); result.Error != nil {
		return result.Error
	} else {
		oldActivities = append(oldActivities, invoice.Activities...)
	}

	for _, act := range oldActivities {
		taskToUpdate[act.TaskID]++
	}

	if err = ReplaceInvoiceActivities(tx, d.ID, d.Activities); err != nil {
		return
	}

	for _, act := range d.Activities {
		taskToUpdate[act.TaskID]++
	}

	// update task user stats
	for uid := range taskToUpdate {
		tasksList = append(tasksList, uid)
	}
	return UpdateUserStats(tx, tasksList)
}
func (d *Invoice) BeforeDelete(tx *gorm.DB) error {
	return ReplaceInvoiceActivities(tx, d.ID, []*Activity{})
}

func (d *Invoice) AfterDelete(tx *gorm.DB) error {

	// update user stats
	taskToUpdate := make(map[uuid.UUID]uint)
	tasksList := make([]uuid.UUID, 0)

	if prevActs, ok := tx.Get("oldActivities"); ok {
		if oldActivities := prevActs.([]*Activity); oldActivities != nil {
			for _, act := range oldActivities {
				taskToUpdate[act.TaskID]++
			}
		}
	}

	for uid := range taskToUpdate {
		tasksList = append(tasksList, uid)
	}
	return UpdateUserStats(tx, tasksList)
}

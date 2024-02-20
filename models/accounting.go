package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"it.terra9/billwise-server/util"
)

type DocumentType int

const (
	_ = iota
	OfferType
	ProformaType
	InvoiceType
	Payment
)

type AccountingDocument struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:(uuid_generate_v4())" json:"id,omitempty"`
	Date           datatypes.Date `json:"date"`
	Status         string         `json:"status"`
	DocumentType   DocumentType   `json:"document_type"`
	DocumentNumber int            `json:"document_number"`
	TaskID         uuid.UUID      `gorm:"not null;" json:"taskId"`
	Task           Task           `gorm:"foreignKey:TaskID"`
	DocumentAmount *float64       `json:"document_amount"`

	// Associations
	Activities []*Activity                   `gorm:"many2many:accountingdocument_activities;constraint:OnDelete:CASCADE;" json:"activities"`
	UserStats  []AccountingDocumentUserStats `json:"user_stats"`

	// Calculated fields
	TotalActivitiesSeconds uint    `json:"total_activities_seconds"`
	TotalActivitiesAmounts float64 `json:"total_activities_amounts"`
}

func (d *AccountingDocument) Count(db *gorm.DB) int64 {
	var totalDocuments int64
	db.Model(&AccountingDocument{}).Count(&totalDocuments)
	return totalDocuments
}

func (d *AccountingDocument) Take(db *gorm.DB, limit int, offset int) interface{} {
	var documents []AccountingDocument
	if limit > 0 {
		db = db.Offset(offset).Limit(limit)
	}
	db.Preload("Task").Preload("Activities.User").Preload("Activities.Task").Order("accounting_documents.date desc, accounting_documents.document_type desc").Find(&documents)
	return documents
}

func GetAccountingDocumentById(db *gorm.DB, id uuid.UUID, userId uuid.UUID) *AccountingDocument {
	accountingDocument := AccountingDocument{}
	if err := accountingDocument.WithUserStats(db, userId).
		Where("accounting_documents.id = ?", id).First(&accountingDocument).Error; err != nil {
		return nil
	}
	return &accountingDocument
}

func (d *AccountingDocument) WithUserStats(db *gorm.DB, userId uuid.UUID) *gorm.DB {
	return db.Model(&AccountingDocument{}).Preload("Task").Preload("Activities").
		Preload("UserStats", "accounting_document_user_stats.user_id = ?", userId)
}

func ReplaceAccountingDocumentActivities(db *gorm.DB, accountingDocumentId uuid.UUID, activities []*Activity) (err error) {

	if err = db.Exec(`DELETE FROM accountingdocument_activities WHERE accounting_document_id = ?`, accountingDocumentId).Error; err != nil {
		return err
	}
	if len(activities) > 0 {
		for _, o := range activities {
			if err = db.Exec(`INSERT INTO "accountingdocument_activities" ("accounting_document_id","activity_id") VALUES (?, ?)`,
				accountingDocumentId, o.ID).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *AccountingDocument) Update(tx *gorm.DB) (err error) {

	return tx.Omit("Activities.*").Preload("User").Preload("Task").Model(d).
		Updates(d).Error
}

func (d *AccountingDocument) AfterCreate(tx *gorm.DB) (err error) {
	return d.AfterSave(tx.Session(&gorm.Session{NewDB: true}).Set("userId", util.SessionUserID(tx).String()))
}

func (d *AccountingDocument) AfterSave(tx *gorm.DB) (err error) {
	userId := util.SessionUserID(tx).String()

	return tx.Session(&gorm.Session{NewDB: true, SkipHooks: true}).
		Set("userId", userId).
		Transaction(func(tx *gorm.DB) (err error) {
			activitiesToUpdate := make(map[uuid.UUID]*Activity)
			tasksToUpdate := make(map[uuid.UUID]int)
			tasksArray := make([]uuid.UUID, 0)
			var originalActivities []*Activity
			if err = tx.Model(d).Association("Activities").Find(&originalActivities); err != nil {
				panic(err)
			}
			for i, act := range d.Activities {
				activitiesToUpdate[act.ID] = d.Activities[i]
				tasksToUpdate[act.TaskID]++
			}
			for i, act := range originalActivities {
				activitiesToUpdate[act.ID] = originalActivities[i]
				tasksToUpdate[act.TaskID]++
			}
			if err = ReplaceAccountingDocumentActivities(tx, d.ID, d.Activities); err != nil {
				panic(err)
			}

			for k := range tasksToUpdate {
				tasksArray = append(tasksArray, k)
			}
			return UpdateUserStats(tx, tasksArray)
		})
}

func (d *AccountingDocument) BeforeDelete(tx *gorm.DB) (err error) {
	if err = ReplaceAccountingDocumentActivities(tx, d.ID, []*Activity{}); err != nil {
		return
	}
	return ReplaceAccountingUserStats(tx, d.ID, []*AccountingDocumentUserStats{})
}

func (d *AccountingDocument) AfterDelete(tx *gorm.DB) (err error) {

	var oldAccountingDocument *AccountingDocument
	if ad, ok := tx.Get("oldAccountingDocument"); ok {
		oldAccountingDocument = ad.(*AccountingDocument)
	}

	// update associated task
	return UpdateUserStats(tx, []uuid.UUID{oldAccountingDocument.TaskID})
}

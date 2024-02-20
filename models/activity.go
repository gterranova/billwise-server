package models

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"it.terra9/billwise-server/util"
)

type ActivityStatus int

const (
	_ ActivityStatus = iota
	OfferSubmitted
	ProformaIssued
	InvoiceIssued
	PaymentMade
	UserCashed
)

var defaultHourlyRate = 150.0

type Activity struct {
	ID                  uuid.UUID            `gorm:"type:uuid;primaryKey;default:(uuid_generate_v4())" json:"id,omitempty"`
	UserID              uuid.UUID            `gorm:"not null" json:"userId"`
	User                User                 `gorm:"foreignKey:UserID"`
	TaskID              uuid.UUID            `gorm:"not null" json:"taskId"`
	Task                Task                 `gorm:"foreignKey:TaskID"`
	AccountingDocuments []AccountingDocument `gorm:"many2many:accountingdocument_activities;constraint:OnDelete:CASCADE;" json:"accounting_documents"`
	InvoiceID           *uuid.UUID           `gorm:"constraint:OnDelete:SET NULL;" json:"invoiceId"`
	Invoice             *Invoice             `gorm:"foreignKey:InvoiceID;constraint:OnDelete:SET NULL;"`
	Description         string               `json:"description"`
	PaymentType         *PaymentType         `json:"payment_type"`
	PaymentAmount       *float64             `json:"payment_amount"`
	SecondsBilled       uint                 `json:"seconds_billed"`
	Date                datatypes.Date       `json:"date"`

	// Calculated fields
	ReferenceDocumentID *uuid.UUID          `gorm:"constraint:OnDelete:SET NULL;" json:"referenceDocumentId"`
	ReferenceDocument   *AccountingDocument `gorm:"foreignKey:ReferenceDocumentID;constraint:OnDelete:SET NULL;" json:"reference_document"`
	Status              ActivityStatus      `json:"status"`
	CalculatedAmount    float64             `json:"calculated_amount"`
}

func (activity *Activity) Count(db *gorm.DB) int64 {
	var totalActivities int64
	db.Model(&Activity{}).Count(&totalActivities)
	return totalActivities
}

func (activity *Activity) Take(db *gorm.DB, limit int, offset int) interface{} {
	var activities []Activity
	if limit > 0 {
		db = db.Offset(offset).Limit(limit)
	}
	db.Preload("User").Preload("Task").Preload("AccountingDocuments").
		Preload("ReferenceDocument").
		Order("activities.date desc").Find(&activities)
	return activities
}

func GetActivityById(db *gorm.DB, id uuid.UUID) *Activity {
	activity := Activity{}
	if err := db.Preload("User").Preload("Task").Preload("AccountingDocuments").
		Preload("ReferenceDocument").
		Where("activities.id = ?", id).First(&activity).Error; err != nil {
		return nil
	}
	return &activity
}

func (activity *Activity) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Statement.Select("UserID", "TaskID", "Description", "PaymentType", "PaymentAmount", "Date", "Status", "SecondsBilled", "CalculatedAmount")
	return nil
}

func (activity *Activity) Create(db *gorm.DB) (err error) {
	return db.Create(activity).Error
}

func (activity *Activity) AfterCreate(tx *gorm.DB) (err error) {
	return activity.AfterSave(tx.Session(&gorm.Session{NewDB: true}).Set("userId", util.SessionUserID(tx).String()))
}

func (activity *Activity) FindReferenceDocument(tx *gorm.DB) *AccountingDocument {
	var refDoc *AccountingDocument

	if err := tx.Raw(`SELECT * FROM accounting_documents
	WHERE accounting_documents.id IN (
	SELECT ada.accounting_document_id FROM accountingdocument_activities ada
		WHERE ada.activity_id = ?
	)`, activity.ID).Scan(&activity.AccountingDocuments).Error; err != nil {
		return nil
	}

	activityStatus := ActivityStatus(0)
	for _, doc := range activity.AccountingDocuments {
		if refDoc == nil || ActivityStatus(doc.DocumentType) > activityStatus {
			refDoc = &doc
		}
	}
	if refDoc != nil {
		return GetAccountingDocumentById(tx, refDoc.ID, util.SessionUserID(tx))
	}
	return nil
}

func (activity *Activity) UpdateCalculatedFields(tx *gorm.DB, activityTask *Task, updateDb bool) (err error) {
	var seconds uint
	var amount float64
	var taskHourlyRate float64
	var activityStatus ActivityStatus
	var refDoc *AccountingDocument
	var refDocID *uuid.UUID

	if refDoc = activity.FindReferenceDocument(tx); refDoc != nil {
		activityStatus = ActivityStatus(refDoc.DocumentType)
		refDocID = &refDoc.ID
	}

	if activityTask != nil && activityTask.PaymentAmount != nil && activityTask.PaymentType != nil && *activityTask.PaymentType == HourlyRate {
		taskHourlyRate = *activityTask.PaymentAmount
	}

	if activity.PaymentType != nil {
		switch *activity.PaymentType {
		case HourlyRate:
			seconds = activity.SecondsBilled
			if taskHourlyRate > 0 {
				amount = taskHourlyRate * (float64(seconds) / 3600)
			} else {
				amount = defaultHourlyRate * (float64(seconds) / 3600)
			}
		case FixedFee:
			if activity.PaymentAmount != nil {
				amount = *activity.PaymentAmount
			}
			if taskHourlyRate > 0 {
				seconds = uint(amount / taskHourlyRate * 3600)
			} else {
				seconds = uint(amount / defaultHourlyRate * 3600)
			}
		}
	}

	if refDoc != nil {
		if refDoc.TotalActivitiesSeconds > 0 {
			amount = float64(seconds) / float64(refDoc.TotalActivitiesSeconds) * *refDoc.DocumentAmount
		}
	}

	activity.ReferenceDocumentID = refDocID
	activity.ReferenceDocument = refDoc
	activity.Status = activityStatus
	activity.SecondsBilled = seconds
	activity.CalculatedAmount = amount

	if updateDb {
		if err = tx.Exec(`UPDATE "activities" SET "reference_document_id"=?,"status"=?,"seconds_billed"=?,"calculated_amount"=? WHERE "id" = ?`, refDocID, activityStatus, seconds, amount, activity.ID).Error; err != nil {
			return
		}
	}

	return nil
}

func (activity *Activity) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Statement.Select("UserID", "TaskID", "Description", "PaymentType", "PaymentAmount", "Date", "SecondsBilled") //"Status", "SecondsBilled", "CalculatedAmount"
	return nil
}

func (activity *Activity) Update(tx *gorm.DB) (err error) {
	return tx.Preload("User").Preload("Task").Model(activity).
		Updates(activity).Error
}

func (activity *Activity) AfterSave(tx *gorm.DB) (err error) {
	// update associated task
	return tx.Session(&gorm.Session{NewDB: true, SkipHooks: true}).Set("userId", util.SessionUserID(tx).String()).Transaction(func(tx *gorm.DB) (err error) {
		return UpdateUserStats(tx, []uuid.UUID{activity.TaskID})
	})
}

func (activity *Activity) BeforeDelete(tx *gorm.DB) (err error) {
	// gorm associations won't work
	return activity.DeleteActivityFromAccountingDocuments(tx)
}

func (activity *Activity) DeleteActivityFromAccountingDocuments(tx *gorm.DB) (err error) {
	var oldActivity *Activity
	if ad, ok := tx.Get("oldActivity"); ok {
		oldActivity = ad.(*Activity)
	}
	// gorm associations won't work
	return tx.Exec(`DELETE FROM accountingdocument_activities WHERE activity_id = ?`, oldActivity.ID).Error
}

func (activity *Activity) AfterDelete(tx *gorm.DB) (err error) {

	var oldActivity *Activity
	if ad, ok := tx.Get("oldActivity"); ok {
		oldActivity = ad.(*Activity)
	}

	// update associated task
	return tx.Session(&gorm.Session{NewDB: true}).Set("userId", util.SessionUserID(tx).String()).Transaction(func(tx *gorm.DB) (err error) {
		task := GetTaskById(tx, oldActivity.TaskID, oldActivity.UserID)
		if task == nil {
			return errors.New("error on loading taks")
		}
		return task.Update(tx.Session(&gorm.Session{SkipHooks: true}))
	})
}

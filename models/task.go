package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"it.terra9/billwise-server/util"
)

type PaymentType int
type PaymentFreq int

const (
	HourlyRate PaymentType = iota
	FixedFee
	TaxableExpense
	TaxExemptExpense
	ContributoUnificato
)

const (
	Monthly PaymentFreq = iota
	OneShot
)

type Task struct {
	ID            uuid.UUID    `gorm:"type:uuid;primaryKey;default:(uuid_generate_v4())" json:"id,omitempty"`
	Code          string       `json:"code,omitempty"`
	Name          string       `json:"name"`
	Description   string       `json:"description,omitempty"`
	PaymentType   *PaymentType `json:"payment_type,omitempty"`
	PaymentFreq   *PaymentFreq `json:"payment_freq,omitempty"`
	PaymentAmount *float64     `json:"payment_amount,omitempty"`
	Notes         string       `json:"notes,omitempty"`
	Archived      bool         `json:"archived,omitempty"`

	// Associations
	Activities []*Activity     `json:"activities,omitempty"`
	UserStats  []TaskUserStats `json:"user_stats"`

	// Calculated fields
	TotalSeconds *uint `json:"total_seconds"`
}

func (task *Task) Count(db *gorm.DB) int64 {
	var totalTasks int64
	db.Model(&Task{}).Count(&totalTasks)
	return totalTasks
}

func (task *Task) Take(db *gorm.DB, limit int, offset int) interface{} {
	var tasks []Task
	if limit > 0 {
		db = db.Offset(offset).Limit(limit)
	}
	db.Preload("Activities.User").Preload("Activities.Task").Preload("Activities.ReferenceDocument").Order("tasks.name asc").Find(&tasks)
	return tasks
}

func GetTaskById(db *gorm.DB, id uuid.UUID, userId uuid.UUID) *Task {
	task := Task{ID: id}
	if err := task.WithUserStats(db.Preload("Activities"), userId).
		First(&task).Error; err != nil {
		return nil
	}
	return &task
}

func (task *Task) WithUserStats(db *gorm.DB, userId uuid.UUID) *gorm.DB {

	return db.Model(&Task{}).Preload("UserStats", "task_user_stats.user_id = ?", userId).
		Where("tasks.archived = ?", false)
}

func (task *Task) Update(tx *gorm.DB) (err error) {
	return tx.Omit("Activities", "UserStats").Model(task).
		Updates(task).Error
}

func (task *Task) AfterSave(tx *gorm.DB) (err error) {
	return tx.Session(&gorm.Session{NewDB: true, SkipHooks: true}).Set("userId", util.SessionUserID(tx).String()).Transaction(func(tx *gorm.DB) (err error) {
		return UpdateUserStats(tx, []uuid.UUID{task.ID})
	})
}

func (task *Task) BeforeDelete(tx *gorm.DB) (err error) {
	var oldTask *Task
	if ad, ok := tx.Get("oldTask"); ok {
		oldTask = ad.(*Task)
	}
	err = tx.Session(&gorm.Session{NewDB: true}).Set("userId", util.SessionUserID(tx).String()).Transaction(func(tx *gorm.DB) (err error) {
		if err = ReplaceTaskUserStats(tx, task.ID, []*TaskUserStats{}); err != nil {
			return
		}
		if err = tx.Session(&gorm.Session{SkipHooks: true}).Model(&Activity{}).Delete(&Activity{}, "activities.task_id = ?", oldTask.ID).Error; err != nil {
			return
		}
		if err = tx.Session(&gorm.Session{SkipHooks: true}).Model(&AccountingDocument{}).Delete(&AccountingDocument{}, "accounting_documents.task_id = ?", oldTask.ID).Error; err != nil {
			return
		}
		return nil
	})
	return err
}

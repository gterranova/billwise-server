package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TaskUserStats struct {
	TaskID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"task_id"`
	UserID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	TotalUserSeconds    uint      `json:"total_user_seconds"`
	DocumentUserBase    float64   `json:"document_user_base"`
	DocumentUserSeconds uint      `json:"document_user_seconds"`
	PayableUserBase     float64   `json:"payable_user_base"`
	PayableUserSeconds  uint      `json:"payable_user_seconds"`
	PaidUserBase        float64   `json:"paid_user_base"`
	PaidUserSeconds     uint      `json:"paid_user_seconds"`
}

func ReplaceTaskUserStats(db *gorm.DB, taskId uuid.UUID, userStats []*TaskUserStats) (err error) {

	if err = db.Exec(`DELETE FROM task_user_stats WHERE task_id = ?`, taskId.String()).Error; err != nil {
		return
	}
	if len(userStats) > 0 {
		return db.Model(&TaskUserStats{}).Create(&userStats).Error
	}
	return nil
}

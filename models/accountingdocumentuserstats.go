package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AccountingDocumentUserStats struct {
	AccountingDocumentID uuid.UUID `gorm:"type:uuid;primaryKey" json:"accounting_document_id"`
	UserID               uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	TotalUserSeconds     uint      `json:"total_user_seconds"`
	TotalUserAmounts     float64   `json:"total_user_amounts"`
	UserPerc             float64   `json:"user_perc"`
}

func ReplaceAccountingUserStats(db *gorm.DB, accountingDocumentId uuid.UUID, userStats []*AccountingDocumentUserStats) (err error) {

	if err = db.Exec(`DELETE FROM accounting_document_user_stats WHERE accounting_document_id = ?`, accountingDocumentId).Error; err != nil {
		return
	}
	if len(userStats) > 0 {
		return db.Model(&AccountingDocumentUserStats{}).
			Create(&userStats).Error
	}
	return nil
}

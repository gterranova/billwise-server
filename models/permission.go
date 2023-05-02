package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Permission struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey;default:(uuid_generate_v4())" json:"id,omitempty"`
	Name string    `gorm:"not null" json:"name"`
}

func (permission Permission) Count(db *gorm.DB) int64 {
	var totalPermissions int64
	db.Model(&Permission{}).Count(&totalPermissions)
	return totalPermissions
}

func (permission Permission) Take(db *gorm.DB, limit int, offset int) interface{} {
	var permissions []Permission
	if limit > 0 {
		db = db.Offset(offset).Limit(limit)
	}
	db.Find(&permissions)
	return permissions
}

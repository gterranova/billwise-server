package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role struct {
	ID          uuid.UUID    `gorm:"type:uuid;primaryKey;default:(uuid_generate_v4())" json:"id,omitempty"`
	Name        string       `gorm:"not null" json:"name"`
	Permissions []Permission `gorm:"many2many:role_permissions" json:"permissions"`
}

func (role Role) Count(db *gorm.DB) int64 {
	var totalRoles int64
	db.Model(&Role{}).Count(&totalRoles)
	return totalRoles
}

func (role Role) Take(db *gorm.DB, limit int, offset int) interface{} {
	var roles []Role
	if limit > 0 {
		db = db.Offset(offset).Limit(limit)
	}
	db.Preload("Permissions").Find(&roles)
	return roles
}

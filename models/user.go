package models

import (
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type CompensationScheme uint

const (
	_                        CompensationScheme = iota
	CompensationCustom                          // N/A
	CompensationJunion                          // N/A
	CompensationAssociate                       // < 80000
	CompensationIntermediate                    // 80000-120000
	CompensationSenior                          // 120000-150000
	CompensationPartner                         // > 150000
)

var levelUpThresholds = [...]uint{0, 0, 0, 80000, 120000, 150000, 0}

type User struct {
	ID         uuid.UUID          `gorm:"type:uuid;primaryKey;default:(uuid_generate_v4())" json:"id,omitempty"`
	FirstName  string             `gorm:"not null" json:"firstName"`
	LastName   string             `gorm:"not null" json:"lastName"`
	Email      string             `gorm:"unique" json:"email"`
	ImageUrl   string             `json:"imageUrl"`
	Password   []byte             `gorm:"not null" json:"-"`
	RoleId     uuid.UUID          `gorm:"not null" json:"roleId"`
	Role       Role               `gorm:"foreignKey:RoleId"`
	Level      CompensationScheme `gorm:"not null;default:0" json:"level"`
	FixedFee   float64            `gorm:"not null;default:0" json:"fixed_fee"`
	Quota      float64            `gorm:"not null" json:"quota"`
	Activities []Activity         `gorm:"foreignKey:UserID" json:"activities"`
}

func (user *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return err
	}
	user.Password = hashedPassword
	return nil
}

func (user *User) VerifyPassword(password string) error {
	return bcrypt.CompareHashAndPassword(user.Password, []byte(password))
}

func (user *User) Count(db *gorm.DB) int64 {
	var totalUsers int64
	db.Model(&User{}).Count(&totalUsers)
	return totalUsers
}

func (user *User) Take(db *gorm.DB, limit int, offset int) interface{} {
	var users []User
	if limit > 0 {
		db = db.Offset(offset).Limit(limit)
	}
	db.Preload("Role").Order("quota desc, last_name asc").Find(&users)
	return users
}

func (user *User) LevelUpThreshold() uint {
	return levelUpThresholds[user.Level]
}

func (user *User) RevenuesToNextLevel(contribRevenues float64) float64 {
	amount := float64(user.LevelUpThreshold())
	if amount > 0 {
		return contribRevenues - amount
	}
	return 0
}

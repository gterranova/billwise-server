package util

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func ParseID(s string) (id uuid.UUID, err error) {
	if id, err = uuid.Parse(s); err != nil {
		return id, err
	}
	return id, nil
}

func SessionUserID(tx *gorm.DB) uuid.UUID {
	var uid interface{}
	var ok bool
	if uid, ok = tx.Get("userId"); !ok {
		panic(errors.New("must provide userId"))
	}
	userId, err := ParseID(uid.(string))
	if err != nil {
		panic(err)
	}
	return userId
}

func NewDBSession(tx *gorm.DB) *gorm.DB {
	return tx.Session(&gorm.Session{NewDB: true}).Set("userId", SessionUserID(tx).String())
}

func NewDBSessionSkipHooks(tx *gorm.DB) *gorm.DB {
	return tx.Session(&gorm.Session{NewDB: true, SkipHooks: true}).Set("userId", SessionUserID(tx).String())
}

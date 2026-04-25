package gormutil

import (
	"errors"

	"gorm.io/gorm"
)

func NilIfNotFound[T any](v *T, err error) (*T, error) {
	if err == nil {
		return v, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return nil, err
}

func FirstOrNil[T any](db *gorm.DB, dest *T) (*T, error) {
	err := db.First(dest).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return dest, err
}

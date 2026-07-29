package listopts

import "gorm.io/gorm"

const defaultPageSize = 10

type Pagination struct {
	Page     int `form:"page,default=1"`
	PageSize int `form:"pageSize,default=10"`
}

func (p Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

func (p Pagination) Scope() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Limit(p.PageSize).Offset(p.Offset())
	}
}

package util

import "gorm.io/gorm"

var MaxPageSize = 100

func Paginate(page, pageSize int) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {

		switch {
		case pageSize > MaxPageSize:
			pageSize = MaxPageSize
		case pageSize <= 0:
			pageSize = 20
		}
		offset := page * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

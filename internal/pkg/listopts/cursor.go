package listopts

import (
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

const CursorTimeLayout = "2006-01-02T15:04:05.000000000Z"

type Cursor struct {
	Cursor string `form:"cursor"`
	Limit  int    `form:"limit"`
}

func EncodeCursor(joinedAt time.Time, id uint) string {
	return joinedAt.UTC().Format(CursorTimeLayout) + "_" + strconv.FormatUint(uint64(id), 10)
}

func (c Cursor) Decode() (time.Time, uint, bool) {
	parts := strings.Split(c.Cursor, "_")
	if len(parts) != 2 {
		return time.Time{}, 0, false
	}
	joinedAt, err := time.Parse(CursorTimeLayout, parts[0])
	if err != nil {
		return time.Time{}, 0, false
	}
	id, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return time.Time{}, 0, false
	}
	return joinedAt, uint(id), true
}

func (c Cursor) Scope(column, idCol string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if value, id, ok := c.Decode(); ok {
			db = db.Where(
				column+" > ? OR ("+column+" = ? AND "+idCol+" > ?)",
				value, value, id,
			)
		}
		return db.Limit(c.Limit + 1)
	}
}

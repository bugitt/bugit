package db

import (
	"time"

	"xorm.io/xorm"
)

type BaseModel struct {
	CreatedUnix int64
	Created     time.Time `xorm:"-" gorm:"-" json:"-"`
	UpdatedUnix int64
	Updated     time.Time `xorm:"-" gorm:"-" json:"-"`
}

func (b *BaseModel) BeforeInsert() {
	b.CreatedUnix = time.Now().Unix()
	b.UpdatedUnix = b.CreatedUnix
}

func (b *BaseModel) BeforeUpdate() {
	b.UpdatedUnix = time.Now().Unix()
}

func (b *BaseModel) AfterSet(colName string, _ xorm.Cell) {
	switch colName {
	case "created_unix":
		b.Created = time.Unix(b.CreatedUnix, 0).Local()
	case "updated_unix":
		b.Updated = time.Unix(b.UpdatedUnix, 0)
	}
}

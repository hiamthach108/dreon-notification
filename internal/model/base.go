package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        string         `gorm:"primaryKey;type:varchar(36)"`
	Metadata  datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
	CreatedBy string         `gorm:"type:varchar(36)"`
	UpdatedBy string         `gorm:"type:varchar(36)"`
}

func (base *BaseModel) BeforeCreate(tx *gorm.DB) (err error) {
	if base.ID == "" {
		uuidv6, err := uuid.NewV6()
		if err != nil {
			return err
		}
		base.ID = uuidv6.String()
	}
	return
}

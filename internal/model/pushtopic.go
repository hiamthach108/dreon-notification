package model

type PushTopic struct {
	BaseModel

	Name        string `gorm:"type:varchar(255);not null;unique"`
	Description string `gorm:"type:text;"`
	IsActive    bool   `gorm:"type:boolean;not null;default:true"`
}

func (PushTopic) TableName() string {
	return "push_topics"
}

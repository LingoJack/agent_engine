package model

import "time"

type TableTool struct {
	ID          int64     `gorm:"column:id;type:integer;primaryKey;autoIncrement" json:"id"`
	ToolID      string    `gorm:"column:tool_id;type:text;not null" json:"toolId"`
	ToolName    string    `gorm:"column:tool_name;type:text" json:"toolName"`
	Description string    `gorm:"column:description;type:text" json:"description"`
	Document    string    `gorm:"column:document;type:text" json:"document"`
	Example     string    `gorm:"column:example;type:text" json:"example"`
	Status      string    `gorm:"column:status;type:text" json:"status"`
	CreateTime  time.Time `gorm:"column:create_time;type:datetime;default:CURRENT_TIMESTAMP" json:"createTime"`
	UpdateTime  time.Time `gorm:"column:update_time;type:datetime;default:CURRENT_TIMESTAMP" json:"updateTime"`
}

func (t *TableTool) TableName() string {
	return "t_tool"
}

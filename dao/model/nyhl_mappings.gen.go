// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

const TableNameNyhlMapping = "nyhl_mappings"

// NyhlMapping mapped from table <nyhl_mappings>
type NyhlMapping struct {
	Location  string `gorm:"column:location;primaryKey" json:"location"`
	SurfaceID int32  `gorm:"column:surface_id;not null" json:"surface_id"`
}

// TableName NyhlMapping's table name
func (*NyhlMapping) TableName() string {
	return TableNameNyhlMapping
}

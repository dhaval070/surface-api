// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

const TableNameLocation = "locations"

// Location mapped from table <locations>
type Location struct {
	ID                  int32   `gorm:"column:id;primaryKey" json:"id"`
	Address1            string  `gorm:"column:address1" json:"address1"`
	Address2            string  `gorm:"column:address2;not null" json:"address2"`
	City                string  `gorm:"column:city" json:"city"`
	Name                string  `gorm:"column:name" json:"name"`
	UUID                string  `gorm:"column:uuid" json:"uuid"`
	RecordingHoursLocal string  `gorm:"column:recording_hours_local;not null" json:"recording_hours_local"`
	PostalCode          string  `gorm:"column:postal_code;not null" json:"postal_code"`
	AllSheetsCount      int32   `gorm:"column:all_sheets_count;not null" json:"all_sheets_count"`
	Longitude           float32 `gorm:"column:longitude;not null" json:"longitude"`
	Latitude            float32 `gorm:"column:latitude;not null" json:"latitude"`
	LogoURL             string  `gorm:"column:logo_url;not null" json:"logo_url"`
	ProvinceID          int32   `gorm:"column:province_id;not null" json:"province_id"`
	VenueStatus         string  `gorm:"column:venue_status;not null" json:"venue_status"`
	Zone                string  `gorm:"column:zone;not null" json:"zone"`
	TotalSurfaces       int32   `gorm:"column:total_surfaces;not null" json:"total_surfaces"`
}

// TableName Location's table name
func (*Location) TableName() string {
	return TableNameLocation
}

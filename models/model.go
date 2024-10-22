package models

import "surface-api/dao/model"

type Config struct {
	DB_DSN string `mapstructure:"DB_DSN"`
	Port   string `mapstructure:"port"`
	Mode   string `mapstructure:"mode"`
}

type SiteLocResult struct {
	Site             string         `gorm:"column:site;not null" json:"site"`
	Location         string         `gorm:"column:location" json:"location"`
	LocationID       int32          `gorm:"column:location_id" json:"location_id"`
	Address          string         `gorm:"column:address" json:"address"`
	MatchType        string         `gorm:"column:match_type" json:"match_type"`
	SurfaceID        int32          `gorm:"column:surface_id;not null" json:"surface_id"`
	LiveBarnLocation model.Location `gorm:"foreignKey:LocationID"`
	LinkedSurface    model.Surface  `gorm:"foreignKey:SurfaceID"`
}

func (*SiteLocResult) TableName() string {
	return "sites_locations"
}

type Mapping struct {
	Site        string `json:"site" gorm:"column:site"`
	Location    string `json:"location" gorm:"column:location"`
	SurfaceID   int    `json:"surface_id" gorm:"column:surface_id"`
	SurfaceName string `gorm:"foreignKey:SurfaceID" gorm:"column:surface_name" json:"surface_name"`
}

type SurfaceResult struct {
	ID         int32  `gorm:"column:id;primaryKey" json:"id"`
	LocationID int32  `gorm:"column:location_id;not null" json:"location_id"`
	Name       string `gorm:"column:name;not null" json:"name"`
	Sports     string `gorm:"column:sports;not null" json:"sports"`

	Location model.Location `gorm:"foreignKey:LocationID"`
}

// TableName Surface's table name
func (*SurfaceResult) TableName() string {
	return "surfaces"
}

type SetSurfaceInput struct {
	Site      string `json:"site"`
	Location  string `json:"location"`
	SurfaceID int32  `json:"surface_id"`
}

type Login struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (Login) TableName() string {
	return "users"
}

type RampLocation struct {
	Rarid       int    `json:"rar_id" gorm:"primaryKey"`
	Name        string `json:"name"`
	Abbr        string `json:"abbr"`
	Address     string `json:"address"`
	City        string `json:"city"`
	Prov        string `json:"prov"`
	Pcode       string `json:"pcode"`
	Country     string `json:"country"`
	MatchType   string `json:"match_type"`
	Location    string `json:"location"`
	SurfaceID   int    `json:"surface_id"`
	SurfaceName string `json:"surface_name"`
}

func (RampLocation) TableName() string {
	return "RAMP_Locations"
}

type SetRampSurfaceID struct {
	RarID     int `json:"rar_id" binding:"required"`
	SurfaceID int `json:"surface_id"`
}

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
	Surface          string         `gorm:"column:surface" json:"surface"`
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
	ID              int32  `gorm:"column:id;primaryKey" json:"id"`
	LocationID      int32  `gorm:"column:location_id;not null" json:"location_id"`
	Name            string `gorm:"column:name;not null" json:"name"`
	Sports          string `gorm:"column:sports;not null" json:"sports"`
	LocationName    string `json:"location_name"`
	LocationCity    string `json:"location_city"`
	LocationAddress string `json:"location_address"`
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
	Rarid        int    `json:"rar_id" gorm:"primaryKey"`
	Location     string `json:"location"`
	Address      string `json:"address"`
	City         string `json:"city"`
	ProvinceName string `json:"province_name"`
	Country      string `json:"country"`
	MatchType    string `json:"match_type"`
	SurfaceID    int    `json:"surface_id"`
	SurfaceName  string `json:"surface_name"`
}

func (RampLocation) TableName() string {
	return "RAMP_Locations"
}

type SetRampSurfaceID struct {
	RarID     int    `json:"rar_id" binding:"required"`
	SurfaceID int    `json:"surface_id"`
	Province  string `json:"province"`
}

type SitesConfigInput struct {
	SiteName             string                 `json:"site_name" binding:"required"`
	DisplayName          *string                `json:"display_name"`
	BaseURL              string                 `json:"base_url" binding:"required"`
	HomeTeam             *string                `json:"home_team"`
	ParserType           string                 `json:"parser_type" binding:"required"`
	ParserConfig         map[string]interface{} `json:"parser_config"`
	Enabled              *bool                  `json:"enabled"`
	ScrapeFrequencyHours *int32                 `json:"scrape_frequency_hours"`
	Notes                *string                `json:"notes"`
}

type SitesConfigResponse struct {
	ID                   int32                  `json:"id"`
	SiteName             string                 `json:"site_name"`
	DisplayName          *string                `json:"display_name"`
	BaseURL              string                 `json:"base_url"`
	HomeTeam             *string                `json:"home_team"`
	ParserType           string                 `json:"parser_type"`
	ParserConfig         map[string]interface{} `json:"parser_config"`
	Enabled              *bool                  `json:"enabled"`
	LastScrapedAt        *string                `json:"last_scraped_at"`
	ScrapeFrequencyHours *int32                 `json:"scrape_frequency_hours"`
	Notes                *string                `json:"notes"`
	CreatedAt            string                 `json:"created_at"`
	UpdatedAt            string                 `json:"updated_at"`
}

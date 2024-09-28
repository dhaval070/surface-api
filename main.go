package main

import (
	"log"
	"net/http"
	"surface-api/dao/model"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB
var cfg Config

func init() {
	gin.SetMode(gin.ReleaseMode)

	viper.SetConfigFile("config.yaml")
	viper.SetDefault("port", "8000")

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	err := viper.Unmarshal(&cfg)

	if err != nil {
		panic(err)
	}

	db, err = gorm.Open(mysql.Open(cfg.DB_DSN))
	if err != nil {
		panic(err)
	}
	log.Println(cfg)
}

func main() {
	r := gin.Default()
	r.Use(cors.Default())

	r.GET("/site-locations/:site", getSiteLoc)
	r.GET("/sites", getSites)
	r.GET("/surfaces", getSurfaces)
	r.POST("/set-surface", setSurface)

	if err := r.Run(":" + cfg.Port); err != nil {
		panic(err)
	}
}

func setSurface(c *gin.Context) {
	var input = &SiteLocResult{}

	if err := c.BindJSON(input); err != nil {
		sendError(c, err)
		return
	}

	var surface = &model.Surface{}
	if err := db.Find(surface, input.SurfaceID).Error; err != nil {
		sendError(c, err)
		return
	}

	input.LocationID = surface.LocationID

	if err := db.Model(input).Where("site=? and location=?", input.Site, input.Location).Select("LocationID", "SurfaceID").Updates(input).Error; err != nil {
		sendError(c, err)
		return
	}
	var result = []SiteLocResult{}

	if err := db.Joins("LinkedSurface").Joins("LiveBarnLocation").Find(&result, "site=?", input.Site).Error; err != nil {
		sendError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func getSurfaces(c *gin.Context) {
	var surfaces = []SurfaceResult{}

	if err := db.Joins("Location").Find(&surfaces).Error; err != nil {
		sendError(c, err)
	}
	c.JSON(http.StatusOK, surfaces)
}

func getSites(c *gin.Context) {
	var sites = []model.Site{}

	if err := db.Find(&sites).Error; err != nil {
		sendError(c, err)
		return
	}
	c.JSON(http.StatusOK, sites)
}

func getSiteLoc(c *gin.Context) {
	site := c.Param("site")
	var result = []SiteLocResult{}

	if err := db.Joins("LinkedSurface").Joins("LiveBarnLocation").Find(&result, "site=?", site).Error; err != nil {
		sendError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func sendError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": err.Error(),
	})
}

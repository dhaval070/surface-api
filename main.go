package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"surface-api/dao/model"
	"surface-api/models"

	"encoding/csv"

	"github.com/astaxie/beego/session"
	_ "github.com/astaxie/beego/session/mysql"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB
var cfg models.Config
var sess *session.Manager
var configFile = flag.String("config", "config.yaml", "Path to config file")

func init() {
	flag.Parse()
	gin.SetMode(gin.ReleaseMode)

	viper.SetConfigFile(*configFile)
	viper.SetDefault("port", "8000")
	viper.SetDefault("mode", "production")

	viper.AutomaticEnv()
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

	db.Exec(`SET SESSION sql_mode=(SELECT REPLACE(@@sql_mode,'ONLY_FULL_GROUP_BY',''))`)

	sess, err = session.NewManager("mysql", &session.ManagerConfig{
		CookieName:      "gosession",
		Gclifetime:      3600,
		ProviderConfig:  cfg.DB_DSN,
		EnableSetCookie: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	go sess.GC()
}

func main() {
	r := gin.Default()

	if cfg.Mode == "local" {
		corsCfg := cors.DefaultConfig()
		corsCfg.AllowCredentials = true
		corsCfg.AllowOrigins = []string{"http://localhost:5173"}
		r.Use(cors.New(corsCfg))
	}
	r.Use(AuthMiddleware)

	r.GET("/site-locations/:site", getSiteLoc)
	r.GET("/mappings/:site", getMappings)
	r.GET("/sites", getSites)
	r.GET("/surfaces", getSurfaces)
	r.POST("/set-surface", setSurface)
	r.POST("/set-mapping", setMapping)
	r.POST("/login", login)
	r.GET("/logout", logout)
	r.GET("/session", checkSession)
	r.GET("/report", downloadReport)
	r.GET("/ramp-mappings/:province", rampMappings)
	r.GET("/ramp-provinces", rampProvinces)
	r.POST("/set-ramp-mapping", SetRampMappings)

	// Sites config CRUD routes
	r.GET("/sites-config", getSitesConfig)
	r.GET("/parser-types", getParserTypes)
	r.GET("/sites-config/:id", getSitesConfigByID)
	r.POST("/sites-config", createSitesConfig)
	r.PUT("/sites-config/:id", updateSitesConfig)
	r.DELETE("/sites-config/:id", deleteSitesConfig)

	if err := r.Run(":" + cfg.Port); err != nil {
		panic(err)
	}
}

func downloadReport(c *gin.Context) {
	query := `SELECT
					e.surface_id,
					l.name location_name,
					s.name surface_name,
					date_format(e.datetime, "%W") dow,
					date_format(min(e.datetime), "%Y-%m-%d %T") start_time,
					date_format(max( date_add(e.datetime, INTERVAL 150 minute)), "%Y-%m-%d %T") end_time
				FROM
				events e JOIN surfaces s on e.surface_id=s.id JOIN locations l on l.id=s.location_id
				GROUP BY l.name, s.name, e.surface_id, date(e.datetime)
				ORDER BY location_name, surface_name, surface_id,dayofweek(e.datetime) `

	dbh, err := db.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}

	result, err := dbh.Query(query)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}

	var surfaceId, locationName, surfaceName, dow, startTime, endTime string
	var b = &bytes.Buffer{}
	w := csv.NewWriter(b)
	w.Write([]string{
		"Surface ID", "Location Name", "Surface Name", "day of week", "start time", "end time",
	})

	for result.Next() {
		if err := result.Scan(&surfaceId, &locationName, &surfaceName, &dow, &startTime, &endTime); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		w.Write([]string{surfaceId, locationName, surfaceName, dow, startTime, endTime})
	}
	w.Flush()

	c.Writer.Header().Add("content-type", "text/csv")
	c.Writer.Header().Add("content-disposition", "attachment;filename=report.csv")
	c.Writer.Write(b.Bytes())
}

func setSurface(c *gin.Context) {
	var input = &models.SiteLocResult{}

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
	var result = []models.SiteLocResult{}

	if err := db.Joins("LinkedSurface").Joins("LiveBarnLocation").Find(&result, "site=?", input.Site).Error; err != nil {
		sendError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func setMapping(c *gin.Context) {
	var input = &models.Mapping{}

	if err := c.BindJSON(input); err != nil {
		sendError(c, err)
		return
	}

	var table = input.Site + "_mappings"
	var surface = &model.Surface{}
	if err := db.Find(surface, input.SurfaceID).Error; err != nil {
		sendError(c, err)
		return
	}

	if err := db.Exec(fmt.Sprintf(`update %s set surface_id=? where location=?`, table), input.SurfaceID, input.Location).Error; err != nil {
		sendError(c, err)
		return
	}
	c.AddParam("site", input.Site)
	getMappings(c)
}

func getSurfaces(c *gin.Context) {
	var surfaces = []models.SurfaceResult{}
	province := c.Query("province")

	var err error

	err = db.Raw(`SELECT
			a.id,
			a.location_id,
			a.name,
			a.sports,
			l.name location_name,
			l.city location_city,
			l.address1 location_address
		FROM
			surfaces a
		INNER JOIN locations l ON a.location_id = l.id
		INNER JOIN provinces p ON l.province_id = p.id
		WHERE p.province_name = ? ORDER BY l.name`, province).Scan(&surfaces).Error

	if err != nil {
		sendError(c, err)
	}
	c.JSON(http.StatusOK, surfaces)
}

func getSites(c *gin.Context) {
	var sites = []model.SitesConfig{}

	if err := db.Order("display_name asc").Find(&sites).Error; err != nil {
		sendError(c, err)
		return
	}
	c.JSON(http.StatusOK, sites)
}

func getSiteLoc(c *gin.Context) {
	site := c.Param("site")
	var result = []models.SiteLocResult{}

	if err := db.Joins("LinkedSurface").Joins("LiveBarnLocation").Find(&result, "site=?", site).Error; err != nil {
		sendError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func getMappings(c *gin.Context) {
	site := c.Param("site")
	var table = site + "_mappings"
	var result = []models.Mapping{}

	err := db.Raw(fmt.Sprintf(`SELECT
		"%s" as site,
		%s.location,
		%s.surface_id,
		surfaces.name as surface_name
		FROM %s
		LEFT JOIN surfaces ON %s.surface_id = surfaces.id`,
		site, table, table, table, table)).Scan(&result).Error

	if err != nil {
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

func login(c *gin.Context) {
	var req = &models.Login{}

	if err := c.BindJSON(req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	hash := sha256.Sum256([]byte(req.Password))
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(hash)))
	base64.StdEncoding.Encode(dst, hash[:])

	if err := db.First(req, "username=?", req.Username).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{
				"error": "Invalid username/password",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if req.Password != string(dst) {
		c.JSON(http.StatusOK, gin.H{
			"error": "Invalid username/password",
		})
		return
	}

	s, _ := c.Get("sess")
	sess := s.(session.Store)
	sess.Set("username", req.Username)

	c.JSON(http.StatusOK, gin.H{
		"username": req.Username,
	})
}

func AuthMiddleware(c *gin.Context) {
	s, err := sess.SessionStart(c.Writer, c.Request)
	if err != nil {
		log.Println("session error", err)
	}
	defer s.SessionRelease(c.Writer)

	url := c.Request.URL.String()
	if url != "/login" && url != "/logout" {
		if s.Get("username") == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Session expired",
			})
			return
		}
	}
	c.Set("sess", s)
	c.Next()
}

func checkSession(c *gin.Context) {
	s, _ := c.Get("sess")
	sess := s.(session.Store)
	username := sess.Get("username")

	c.JSON(http.StatusOK, gin.H{
		"username": username,
	})
}

func logout(c *gin.Context) {
	sess.SessionDestroy(c.Writer, c.Request)
	c.Status(http.StatusOK)
}

func rampMappings(c *gin.Context) {
	var province = c.Param("province")
	var result []models.RampLocation

	err := db.Raw(`SELECT
		a.rarid,
		a.name location, a.address, a.city, a.province_name, a.country, a.match_type,
		a.surface_id,
		b.name surface_name
		FROM RAMP_Locations a
		LEFT JOIN locations c ON c.id = a.location_id
		LEFT JOIN surfaces b ON b.id = a.surface_id
		WHERE a.province_name=?`, province).Scan(&result).Error

	if err != nil {
		sendError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func rampProvinces(c *gin.Context) {
	var result []sql.NullString

	err := db.Raw("select distinct(province_name) provinces from RAMP_Locations order by provinces").Scan(&result).Error

	if err != nil {
		sendError(c, err)
		return
	}
	var data []string
	for _, v := range result {
		if v.Valid {
			data = append(data, v.String)
		}
	}
	c.JSON(http.StatusOK, data)
}

func SetRampMappings(c *gin.Context) {
	var input = &models.SetRampSurfaceID{}

	if err := c.BindJSON(input); err != nil {
		sendError(c, err)
		return
	}

	var rec = &models.RampLocation{}

	if err := db.First(rec, "rarid=?", input.RarID).Error; err != nil {
		sendError(c, err)
		return
	}

	if err := db.Exec(`UPDATE RAMP_Locations SET surface_id=? where rarid=?`, input.SurfaceID, input.RarID).Error; err != nil {
		sendError(c, err)
		return
	}

	c.AddParam("province", input.Province)
	rampMappings(c)
}

// getSitesConfig retrieves all sites config records
func getSitesConfig(c *gin.Context) {
	var sitesConfigs []model.SitesConfig

	if err := db.Find(&sitesConfigs).Error; err != nil {
		sendError(c, err)
		return
	}

	var response []models.SitesConfigResponse
	for _, sc := range sitesConfigs {
		response = append(response, convertToSitesConfigResponse(sc))
	}

	c.JSON(http.StatusOK, response)
}

// getSitesConfigByID retrieves a single sites config record by ID
func getSitesConfigByID(c *gin.Context) {
	id := c.Param("id")
	var sitesConfig model.SitesConfig

	if err := db.First(&sitesConfig, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sites config not found"})
			return
		}
		sendError(c, err)
		return
	}

	c.JSON(http.StatusOK, convertToSitesConfigResponse(sitesConfig))
}

// createSitesConfig creates a new sites config record
func createSitesConfig(c *gin.Context) {
	var input models.SitesConfigInput

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sitesConfig := convertToSitesConfigModel(input)

	if err := db.Create(&sitesConfig).Error; err != nil {
		sendError(c, err)
		return
	}

	c.JSON(http.StatusCreated, convertToSitesConfigResponse(sitesConfig))
}

// updateSitesConfig updates an existing sites config record
func updateSitesConfig(c *gin.Context) {
	id := c.Param("id")
	var input models.SitesConfigInput

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var sitesConfig model.SitesConfig
	if err := db.First(&sitesConfig, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sites config not found"})
			return
		}
		sendError(c, err)
		return
	}

	updatedConfig := convertToSitesConfigModel(input)
	updatedConfig.ID = sitesConfig.ID
	updatedConfig.CreatedAt = sitesConfig.CreatedAt

	if err := db.Save(&updatedConfig).Error; err != nil {
		sendError(c, err)
		return
	}

	c.JSON(http.StatusOK, convertToSitesConfigResponse(updatedConfig))
}

// deleteSitesConfig deletes a sites config record
func deleteSitesConfig(c *gin.Context) {
	id := c.Param("id")
	var sitesConfig model.SitesConfig

	if err := db.First(&sitesConfig, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sites config not found"})
			return
		}
		sendError(c, err)
		return
	}

	if err := db.Delete(&sitesConfig).Error; err != nil {
		sendError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sites config deleted successfully"})
}

// Helper function to convert model to response
func convertToSitesConfigResponse(sc model.SitesConfig) models.SitesConfigResponse {
	response := models.SitesConfigResponse{
		ID:         sc.ID,
		SiteName:   sc.SiteName,
		BaseURL:    sc.BaseURL,
		ParserType: sc.ParserType,
		CreatedAt:  sc.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:  sc.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	if sc.DisplayName.Valid {
		response.DisplayName = &sc.DisplayName.String
	}
	if sc.HomeTeam.Valid {
		response.HomeTeam = &sc.HomeTeam.String
	}
	if sc.ParserConfig != nil {
		response.ParserConfig = *sc.ParserConfig
	}
	if sc.Enabled.Valid {
		response.Enabled = &sc.Enabled.Bool
	}
	if sc.LastScrapedAt.Valid {
		lastScraped := sc.LastScrapedAt.Time.Format("2006-01-02 15:04:05")
		response.LastScrapedAt = &lastScraped
	}
	if sc.ScrapeFrequencyHours.Valid {
		response.ScrapeFrequencyHours = &sc.ScrapeFrequencyHours.Int32
	}
	if sc.Notes.Valid {
		response.Notes = &sc.Notes.String
	}

	return response
}

// Helper function to convert input to model
func convertToSitesConfigModel(input models.SitesConfigInput) model.SitesConfig {
	sitesConfig := model.SitesConfig{
		SiteName:   input.SiteName,
		BaseURL:    input.BaseURL,
		ParserType: input.ParserType,
	}

	if input.DisplayName != nil {
		sitesConfig.DisplayName = sql.NullString{String: *input.DisplayName, Valid: true}
	}
	if input.HomeTeam != nil {
		sitesConfig.HomeTeam = sql.NullString{String: *input.HomeTeam, Valid: true}
	}
	if input.ParserConfig != nil {
		pc := model.ParserConfig(input.ParserConfig)
		sitesConfig.ParserConfig = &pc
	}
	if input.Enabled != nil {
		sitesConfig.Enabled = sql.NullBool{Bool: *input.Enabled, Valid: true}
	}
	if input.ScrapeFrequencyHours != nil {
		sitesConfig.ScrapeFrequencyHours = sql.NullInt32{Int32: *input.ScrapeFrequencyHours, Valid: true}
	}
	if input.Notes != nil {
		sitesConfig.Notes = sql.NullString{String: *input.Notes, Valid: true}
	}

	return sitesConfig
}

// getParserTypes returns all available parser types
func getParserTypes(c *gin.Context) {
	parserTypes := []string{
		"day_details",
		"day_details_parser1",
		"day_details_parser2",
		"month_based",
		"group_based",
		"custom",
		"external",
	}

	c.JSON(http.StatusOK, parserTypes)
}

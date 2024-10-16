package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
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

func init() {
	gin.SetMode(gin.ReleaseMode)

	viper.SetConfigFile("config.yaml")
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
	r.GET("/sites", getSites)
	r.GET("/surfaces", getSurfaces)
	r.POST("/set-surface", setSurface)
	r.POST("/login", login)
	r.GET("/logout", logout)
	r.GET("/session", checkSession)
	r.GET("/report", downloadReport)

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
					date_format(min(e.datetime), "%Y-%m-%d %H:%I:%S") start_time,
					date_format(max( date_add(e.datetime, INTERVAL 90 minute)), "%Y-%m-%d %H:%I:%S") end_time
				FROM
				events e JOIN surfaces s on e.surface_id=s.id JOIN locations l on l.id=s.location_id
				GROUP BY location_name, surface_name, surface_id, date(e.datetime)
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

func getSurfaces(c *gin.Context) {
	var surfaces = []models.SurfaceResult{}

	if err := db.Order("Location.Name,name").Joins("Location").Find(&surfaces).Error; err != nil {
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
	var result = []models.SiteLocResult{}

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

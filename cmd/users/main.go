package main

import (
	"crypto/sha256"
	"encoding/base64"
	"log"
	"surface-api/models"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	username string
	password string
	cfg      models.Config
	db       *gorm.DB
)

var root = &cobra.Command{
	Use:   "user",
	Short: "Manager user",
}

var userCmd = &cobra.Command{
	Use:   "create",
	Short: "create user",
	RunE: func(cmd *cobra.Command, args []string) error {
		hash := sha256.Sum256([]byte(password))
		dst := make([]byte, base64.StdEncoding.EncodedLen(len(hash)))
		base64.StdEncoding.Encode(dst, hash[:])

		err := db.Create(&models.Login{
			Username: username,
			Password: string(dst),
		}).Error
		return err
	},
}

func init() {
	userCmd.Flags().StringVar(&username, "username", "", "username")
	userCmd.Flags().StringVar(&password, "password", "", "username")
	userCmd.MarkFlagRequired("username")
	userCmd.MarkFlagRequired("password")
	root.AddCommand(userCmd)

	viper.SetConfigFile("config.yaml")
	viper.SetDefault("mode", "production")

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
}

func main() {
	if err := root.Execute(); err != nil {
		log.Fatal(err)
	}
}

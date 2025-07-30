package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/EPecherkin/catty-counting/config"
	// "gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func NewConnection() (*gorm.DB, error) {
	// TODO: need JSON format
	logLevel := gormlogger.Warn
	if config.LogDebug() {
		logLevel = gormlogger.Info
	}
	dblgr := gormlogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		gormlogger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			// ParameterizedQueries:      true,          // Don't include params in the SQL log
			// Colorful:                  false,         // Disable color
		},
	)

	// dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
	// 	os.Getenv("DB_HOSTNAME"),
	// 	os.Getenv("DB_USERNAME"),
	// 	os.Getenv("DB_PASSWORD"),
	// 	os.Getenv("DB_NAME"),
	// 	os.Getenv("DB_PORT"),
	// )
	// db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: dblgr})

	db, err := gorm.Open(sqlite.Open("tmp/dev.sqlite3"), &gorm.Config{Logger: dblgr})
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", errors.WithStack(err))
	}

	err = db.AutoMigrate(&User{}, &Chat{}, &Message{}, &File{}, &ExposedFile{})
	if err != nil {
		return nil, fmt.Errorf("auto-migrating database: %w", errors.WithStack(err))
	}

	return db, nil
}

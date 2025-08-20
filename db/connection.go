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
	logger := gormlogger.New(
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
	// db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger})

	db, err := gorm.Open(sqlite.Open("tmp/dev.sqlite3"), &gorm.Config{Logger: logger})
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", errors.WithStack(err))
	}

	if err := db.AutoMigrate(&User{}, &Chat{}, &Message{}, &File{}, &ExposedFile{}, &Receipt{}, &Product{}, &Category{}, &ProductCategory{}); err != nil {
		return nil, fmt.Errorf("auto-migrating database: %w", errors.WithStack(err))
	}

	if err := seed(db); err != nil {
		return nil, fmt.Errorf("seeding database: %w", errors.WithStack(err))
	}

	return db, nil
}

func seed(db *gorm.DB) error {
	categories := []Category{
		{Title: "Housing", Details: "Mortgage or rent;Property taxes;Household repairs;HOA fees"},
		{Title: "Transportation", Details: "Car payment;Car warranty;Gas;Tires;Maintenance and oil changes;Parking fees;Repairs;Registration and DMV Fees"},
		{Title: "Food", Details: "Groceries;Restaurants;Pet food"},
		{Title: "Utilities", Details: "Electricity;Water;Garbage;Phones;Cable;Internet"},
		{Title: "Clothing", Details: "Adults’ clothing;Adults’ shoes;Children’s clothing;Children’s shoes"},
		{Title: "Medical/Healthcare", Details: "Primary care;Dental care;Specialty care (dermatologists, orthodontics, optometrists, etc.);Urgent care;Medications;Medical devices"},
		{Title: "Insurance", Details: "Health insurance;Homeowner’s or renter’s insurance;Home warranty or protection plan;Auto insurance;Life insurance;Disability insurance"},
		{Title: "Household Items/Supplies", Details: "Toiletries;Laundry detergent;Dishwasher detergent;Cleaning supplies;Tools"},
		{Title: "Personal", Details: "Gym memberships;Haircuts;Salon services;Cosmetics (like makeup or services like laser hair removal);Babysitter;Subscriptions"},
		{Title: "Debt", Details: "Personal loans;Student loans;Credit cards"},
		{Title: "Retirement", Details: "Financial planning;Investing"},
		{Title: "Education", Details: "Children’s college;Your college;School supplies;Books"},
		{Title: "Savings", Details: "Emergency fund;Big purchases like a new mattress or laptop;Other savings"},
		{Title: "Gifts/Donations", Details: "Birthday;Anniversary;Wedding;Christmas;Special occasion;Charities"},
		{Title: "Entertainment", Details: "Alcohol and/or bars;Games;Movies;Concerts;Vacations;Subscriptions (Netflix, Amazon, Hulu, etc.)"},
	}

	var existingCategories []Category
	db.Find(&existingCategories)

	for _, category := range categories {
		if err := db.First(&Category{}, &Category{Title: category.Title}).Error; err == gorm.ErrRecordNotFound {
			if err := db.Create(&category).Error; err != nil {
				return fmt.Errorf("seeding categories: %w", errors.WithStack(err))
			}
		}
	}
	return nil
}

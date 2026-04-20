package infrastructure

import (
	"clean-architecture/pkg/framework"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Database modal
type Database struct {
	*gorm.DB
}

// NewDatabase creates a new database instance
func NewDatabase(logger framework.Logger, env *framework.Env) Database {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		env.DBHost,
		env.DBUsername,
		env.DBPassword,
		env.DBName,
		env.DBPort,
	)

	logger.Info("opening db connection")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.GetGormLogger()})
	if err != nil {
		logger.Panic(err)
	}

	conn, err := db.DB()
	if err != nil {
		logger.Info("couldn't get db connection")
		logger.Panic(err)
	}

	conn.SetConnMaxLifetime(time.Minute * 5)
	conn.SetMaxOpenConns(5)
	conn.SetMaxIdleConns(1)

	return Database{DB: db}
}

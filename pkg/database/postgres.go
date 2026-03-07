package database

import (
	"fmt"

	"github.com/hiamthach108/dreon-notification/config"
	"github.com/hiamthach108/dreon-notification/internal/model"
	"github.com/hiamthach108/dreon-notification/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDbClient(config *config.AppConfig, logger logger.ILogger) (*gorm.DB, error) {
	dialector := getPostgresSQLDialector(
		config.Postgres.ConnectionName,
		config.Postgres.Host,
		config.Postgres.Port,
		config.Postgres.Username,
		config.Postgres.Password,
		config.Postgres.DBName,
		config.Postgres.SSL,
	)
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(config.Postgres.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.Postgres.MaxOpenConns)

	// ping to verify connection
	if err := sqlDB.Ping(); err != nil {
		logger.Error("Failed to ping database", "error", err)
		return nil, err
	}

	// Auto migrate your models here if needed
	if err := autoMigration(db, logger); err != nil {
		return nil, err
	}

	logger.Info("Connected to PostgreSQL database successfully")
	return db, nil
}

func getPostgresSQLDialector(connectionName string, host string, port int,
	username string, password string, dbname string, ssl bool) gorm.Dialector {

	sslmode := "disable"

	if ssl {
		sslmode = "require"
	}

	if connectionName != "" {
		dsn := fmt.Sprintf(
			"host=%s user=%s dbname=%s password=%s sslmode=%s",
			connectionName, username, dbname, password, sslmode,
		)
		return postgres.New(postgres.Config{
			DriverName: "cloudsqlpostgres",
			DSN:        dsn,
		})
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, port, username, password, dbname, sslmode,
	)
	return postgres.Open(dsn)
}

func autoMigration(db *gorm.DB, logger logger.ILogger) error {
	logger.Info("Starting database auto migration")

	if err := db.AutoMigrate(
		&model.Notification{},
	); err != nil {
		logger.Error("Failed to auto migrate database", "error", err)
		return err
	}

	return nil
}

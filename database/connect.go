package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"it.terra9/billwise-server/models"
	"it.terra9/billwise-server/util"

	sqliteGo "github.com/mattn/go-sqlite3"
	uuid "github.com/satori/go.uuid"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func uuid_generate_v4(arguments ...interface{}) (string, error) {
	return uuid.NewV4().String(), nil // Return a string value.
}

func Connect() {
	var err error
	var database *gorm.DB

	//dsn := "root:admin@tcp(127.0.0.1:3306)/billwise?charset=utf8mb4&parseTime=True&loc=Local"
	//database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	dbDriver := util.Config.DBDriver
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Silent, // Log level
			IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,          // Don't include params in the SQL log
			Colorful:                  true,          // Disable color
		},
	)
	gormConfig := &gorm.Config{
		Logger: newLogger,
	}
	switch dbDriver {
	case "mysql":
		DBURL := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local", util.Config.DBUser, util.Config.DBPassword, util.Config.DBHost, util.Config.DBPort, util.Config.DBName)
		database, err = gorm.Open(mysql.Open(DBURL), gormConfig)
		if err != nil {
			fmt.Printf("Cannot connect to %s database\n", dbDriver)
			log.Fatal("This is the error:", err)
		} else {
			fmt.Printf("We are connected to the %s database\n", dbDriver)
		}
		database.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")

	case "postgres":
		DBURL := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable password=%s", util.Config.DBHost, util.Config.DBPort, util.Config.DBUser, util.Config.DBName, util.Config.DBPassword)
		database, err = gorm.Open(postgres.Open(DBURL), gormConfig)
		if err != nil {
			fmt.Printf("Cannot connect to %s database\n", dbDriver)
			log.Fatal("This is the error:", err)
		} else {
			fmt.Printf("We are connected to the %s database\n", dbDriver)
		}
		database.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")

	case "sqlite":
		const CustomDriverName = "sqlite3_extended"

		sql.Register(CustomDriverName,
			&sqliteGo.SQLiteDriver{
				ConnectHook: func(conn *sqliteGo.SQLiteConn) error {
					err := conn.RegisterFunc(
						"uuid_generate_v4",
						uuid_generate_v4,
						true,
					)
					return err
				},
			},
		)
		var conn *sql.DB
		conn, err = sql.Open(CustomDriverName, "billwise.db")
		if err != nil {
			panic(err)
		}
		conn.SetMaxOpenConns(1)

		database, err = gorm.Open(sqlite.Dialector{
			DriverName: CustomDriverName,
			DSN:        "billwise.db",
			Conn:       conn,
		}, &gorm.Config{
			Logger:                   logger.Default.LogMode(logger.Error),
			SkipDefaultTransaction:   true,
			DisableNestedTransaction: true,
		})
	}

	if err != nil {
		panic("failed to connect to the database")
	}

	if err = database.AutoMigrate(
		&models.Permission{},
		&models.Role{},
		&models.User{}, &models.Task{},
		&models.Activity{},
		&models.AccountingDocument{},
		&models.Invoice{},
		&models.TaskUserStats{},
		&models.AccountingDocumentUserStats{}); err == nil && database.Migrator().HasTable(&models.User{}) {
		if err := database.First(&models.User{}).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			//Insert seed data
			initialize(database)
		}
	}
	DB = database
}

func initialize(database *gorm.DB) {
	permissions := make([]models.Permission, 7)
	contexts := [7]string{
		"users", "roles", "permissions", "tasks",
		"activities", "accounting", "invoices",
	}
	for i, context := range contexts {
		if err := database.Where(models.Permission{
			Name: fmt.Sprintf("edit_%v", context),
		}).Assign(models.Permission{
			Name: fmt.Sprintf("edit_%v", context),
		}).FirstOrCreate(&permissions[i]).Error; err != nil {
			panic(err)
		}
	}

	admin_role := models.Role{
		Name:        "admin",
		Permissions: permissions,
	}
	database.Create(&admin_role)

	admin_user := models.User{
		FirstName: util.Config.DefaultAdminName,
		LastName:  util.Config.DefaultAdminLastname,
		Email:     util.Config.DefaultAdminEmail,
		ImageUrl:  "",
		Role:      admin_role,
		Quota:     0.5,
	}
	admin_user.SetPassword(util.Config.DefaultUserPassword)

	database.Create(&admin_user)
}

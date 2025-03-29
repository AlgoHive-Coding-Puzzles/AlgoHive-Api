package database

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"api/config"
	"api/metrics"
	"api/models"
	"api/utils"
	"api/utils/permissions"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

var AdminRole = "Owner"
var DefaultPassword = "admin"

// InitDB initializes the database connection and migrates the models and populates the database with default values if needed
func InitDB() {
    dsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable TimeZone=Europe/Paris", config.PostgresHost, config.PostgresPort, config.PostgresUser, config.PostgresDB, config.PostgresPassword)
    
    	// Create a custom logger for GORM that records metrics
	newLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: false,
			Colorful:                  true,
		},
	)

    var gormConfig *gorm.Config
    if config.Env == "production" {
        gormConfig = &gorm.Config{
            Logger: newLogger,
        }
    } else {
        gormConfig = &gorm.Config{}
    }

    var err error
    DB, err = gorm.Open(postgres.Open(dsn), gormConfig)
    if err != nil {
        log.Fatal("failed to connect database: ", err)
    }

    err = DB.AutoMigrate(
        &models.User{},
        &models.Role{},
        &models.PasswordReset{},
        &models.Catalog{},
        &models.Scope{},
        &models.Group{},
        &models.Competition{},
        &models.Try{},
    )

    Populate()
    if err != nil {
        log.Fatal("failed to migrate database: ", err)
    }

    // Add callback for metrics collection
	err = DB.Callback().Create().Before("gorm:create").Register("metrics:create_before", func(db *gorm.DB) {
		db.InstanceSet("start_time", time.Now())
	})
	if err != nil {
		log.Printf("Error registering callback: %v", err)
	}

	err = DB.Callback().Create().After("gorm:create").Register("metrics:create_after", func(db *gorm.DB) {
		if startTime, ok := db.InstanceGet("start_time"); ok {
			metrics.RecordDBOperation("create", db.Statement.Table, startTime.(time.Time))
		}
	})
	if err != nil {
		log.Printf("Error registering callback: %v", err)
	}

	// Similar callbacks for Query, Update and Delete operations...
	// Add for Query
	DB.Callback().Query().Before("gorm:query").Register("metrics:query_before", func(db *gorm.DB) {
		db.InstanceSet("start_time", time.Now())
	})
	DB.Callback().Query().After("gorm:query").Register("metrics:query_after", func(db *gorm.DB) {
		if startTime, ok := db.InstanceGet("start_time"); ok {
			metrics.RecordDBOperation("query", db.Statement.Table, startTime.(time.Time))
		}
	})

	// Add for Update
	DB.Callback().Update().Before("gorm:update").Register("metrics:update_before", func(db *gorm.DB) {
		db.InstanceSet("start_time", time.Now())
	})
	DB.Callback().Update().After("gorm:update").Register("metrics:update_after", func(db *gorm.DB) {
		if startTime, ok := db.InstanceGet("start_time"); ok {
			metrics.RecordDBOperation("update", db.Statement.Table, startTime.(time.Time))
		}
	})

	// Add for Delete
	DB.Callback().Delete().Before("gorm:delete").Register("metrics:delete_before", func(db *gorm.DB) {
		db.InstanceSet("start_time", time.Now())
	})
	DB.Callback().Delete().After("gorm:delete").Register("metrics:delete_after", func(db *gorm.DB) {
		if startTime, ok := db.InstanceGet("start_time"); ok {
			metrics.RecordDBOperation("delete", db.Statement.Table, startTime.(time.Time))
		}
	})
}

// Populate populates the database with default values if needed
func Populate() {
    var countRole, countUser int64
    var adminRole models.Role

    // Check if there is no role and no user in the database
    DB.Model(&models.Role{}).Count(&countRole)
    DB.Model(&models.User{}).Count(&countUser)
    if countRole == 0 && countUser == 0 {
        // Create default role admin
        adminRole = models.Role{Name: AdminRole, Permissions: permissions.GetAdminPermissions()}
        DB.Create(&adminRole)
        log.Println("Default role admin created")

        // Create default user admin with a default hashed password either from the .env file or the DefaultPassword constant
        password := DefaultPassword
        if config.DefaultPassword != "" {
            password = config.DefaultPassword
        }

        password, err := utils.HashPassword(password)
        if err != nil {
            panic(err)
        }
        
        user := models.User{
            Email:    "admin@admin.com",
            Firstname: "Admin",
            Lastname:  "Admin",
            Password: password,
            LastConnected: nil,
            Roles: []*models.Role{&adminRole},
        }
        DB.Create(&user)
        log.Println("Default user admin created")
    }

       // Check if there is no API Environment in the database
        var countCatalog int64
        DB.Model(&models.Catalog{}).Count(&countCatalog)
        
        beeApis := config.BeeApis
        if beeApis != "" {    
            catalogsList := strings.Split(beeApis, ",")
    
            // If count is different or we need to sync
            if countCatalog != int64(len(catalogsList)) {
                for _, catalogAddress := range catalogsList {
                    // Make a GET catalog/name to get the catalog name
                    resp, err := http.Get(fmt.Sprintf("%s/name", catalogAddress))
                    if err != nil {
                        log.Println("Error while getting the catalog name: ", err)
                        continue
                    }
                    defer resp.Body.Close()
    
                    if resp.StatusCode != http.StatusOK {
                        log.Println("Error while getting the catalog name: ", resp.Status)
                        continue
                    }
    
                    body, err := io.ReadAll(resp.Body)
                    if err != nil {
                        log.Println("Error while reading the catalog name: ", err)
                        continue
                    }
    
                    var result map[string]string
                    err = json.Unmarshal(body, &result)
                    if err != nil {
                        log.Println("Error while unmarshalling the catalog name: ", err)
                        continue
                    }
                    
                    catalogName, nameOk := result["name"]
                    catalogDesc, descOk := result["description"]
                    if !nameOk || !descOk {
                        log.Println("Error while getting the catalog name or description: key not found")
                        continue
                    }
    
                    // Check if a catalog with this name already exists
                    var existingCatalog models.Catalog
                    if err := DB.Where("name = ?", catalogName).First(&existingCatalog).Error; err == nil {
                        // Update existing catalog
                        existingCatalog.Description = catalogDesc
                        existingCatalog.Address = catalogAddress
                        DB.Save(&existingCatalog)
                    } else {
                        // Create new catalog
                        newCatalog := models.Catalog{Name: catalogName, Description: catalogDesc, Address: catalogAddress}
                        DB.Create(&newCatalog)
                    }
                }
            }
        }
}
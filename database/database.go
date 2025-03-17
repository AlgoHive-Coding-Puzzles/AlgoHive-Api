package database

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"api/config"
	"api/models"
	"api/utils"
	"api/utils/permissions"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

var AdminRole = "Owner"
var DefaultPassword = "admin"

// InitDB initializes the database connection and migrates the models and populates the database with default values if needed
func InitDB() {
    dsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable TimeZone=Europe/Paris", config.PostgresHost, config.PostgresPort, config.PostgresUser, config.PostgresDB, config.PostgresPassword)
    
    var err error
    DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("failed to connect database: ", err)
    }

    err = DB.AutoMigrate(
        &models.User{},
        &models.Role{},
        // &models.Input{},
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
    var catalogsCount int64
    DB.Model(&models.Catalog{}).Count(&catalogsCount)
    catalogs := config.BeeApis
    if catalogsCount == 0 && catalogs != "" {
        catalogsList := strings.Split(catalogs, ",")
        for _, catalog := range catalogsList {
            log.Println("Creating Catalog from API Environment: ", catalog)
            // Make a GET catalog/name to get the catalog name if it ends up in an error just ignore it
            resp, err := http.Get(fmt.Sprintf("%s/name", catalog))
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
                log.Println("Error while reading the catalogc name: ", err)
                continue
            }

            var result map[string]string
            err = json.Unmarshal(body, &result)
            if err != nil {
                log.Println("Error while unmarshalling the catalog name: ", err)
                continue
            }
            catatalogName, nameOk := result["name"]
            catatalogDesc, descOk := result["description"]
            if !nameOk || !descOk {
                log.Println("Error while getting the catalog name or description: key not found")
                continue
            }

            newCatalog := models.Catalog{Name: catatalogName, Description: catatalogDesc, Address: catalog}
            DB.Create(&newCatalog)
            log.Println("API Environment created: ", catatalogName)
        }
    }
}
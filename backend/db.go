package main

import (
	  "gorm.io/driver/postgres"
	  "gorm.io/gorm"
	  "log"
	  "os"
	  "fmt"
)

var DB *gorm.DB

// User model
// User and Device types are now in types.go
// Remove duplicate type definitions

func InitDB() error {
	   // Log all environment variables for debugging
	   fmt.Println("Loaded environment variables:")
	   for _, env := range os.Environ() {
			   fmt.Println(env)
	   }
	   dsn := os.Getenv("POSTGRES_DSN")
	   if dsn == "" {
			   // Try to join multiline DSN from .env
			   dsnLines := []string{}
			   for _, key := range []string{"host", "user", "password", "dbname", "port", "sslmode"} {
					   v := os.Getenv(key)
					   if v != "" {
							   dsnLines = append(dsnLines, key+"="+v)
					   }
			   }
			   if len(dsnLines) > 0 {
					   dsn = "" + dsnLines[0]
					   for _, part := range dsnLines[1:] {
							   dsn += " " + part
					   }
			   } else {
					   dsn = "host=localhost user=postgres password=postgres dbname=tsslib port=5432 sslmode=disable"
			   }
	   }
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		return err
	}
	DB = db
	// Auto-migrate tables
   if err := db.AutoMigrate(&User{}, &Device{}); err != nil {
		log.Printf("Failed to migrate database: %v", err)
		return err
	}
	return nil
}

// Save key share for a device
func SaveDeviceKeyShare(deviceID string, keyShare []byte) error {
	   return DB.Model(&Device{}).Where("id = ?", deviceID).Update("key_share_json", keyShare).Error
}

// Load key share for a device
func LoadDeviceKeyShare(deviceID string) ([]byte, error) {
	   var device Device
	   if err := DB.First(&device, "id = ?", deviceID).Error; err != nil {
			   return nil, err
	   }
	   return device.KeyShareJSON, nil
}

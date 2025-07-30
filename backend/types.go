package main

// User model for GORM
// Add fields as needed (e.g., Password, Threshold)
type User struct {
	ID       string `gorm:"primaryKey"`
	Username string
	Password string // Add if needed
	Threshold int   // Add if needed
	Devices  []Device `gorm:"foreignKey:UserID"`
}

type Device struct {
	ID     string `gorm:"primaryKey"`
	UserID string
	Name   string
	KeyShareJSON []byte // Store key share as JSON
}

type EncryptedMessage struct {
	ID         string
	UserID     string
	Ciphertext []byte
	SessionID  string
}

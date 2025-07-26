package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func InitDB() error {
	var err error
	db, err = sql.Open("sqlite3", "nokey.db")
	if err != nil {
		return err
	}
	// Create tables if not exist
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT,
		password TEXT
	);
	CREATE TABLE IF NOT EXISTS devices (
		id TEXT PRIMARY KEY,
		user_id TEXT,
		name TEXT
	);
	CREATE TABLE IF NOT EXISTS key_shares (
		id TEXT PRIMARY KEY,
		user_id TEXT,
		device_id TEXT,
		key_share TEXT
	);`)
	return err
}

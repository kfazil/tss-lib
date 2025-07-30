package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// Removed duplicate InitDB. Use the one in db.go

package db

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB() {
	var err error
	DB, err = sql.Open("sqlite", "chat.db")
	if err != nil {
		log.Fatal(err)
	}
	if err = DB.Ping(); err != nil {
		log.Fatal(err)
	}
	log.Println("SQLite connected")
}

func CreateTables() {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE,
		password TEXT,
		role TEXT DEFAULT 'user'
	);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sender TEXT,
		text TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := DB.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

func GetUserID(username string) (int, error) {
	var id int

	err := DB.QueryRow(
		"SELECT id FROM users WHERE username=?",
		username,
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}
func GetUserRole(username string) (string, error) {
	var role string
	err := DB.QueryRow(
		"SELECT role FROM users WHERE username=?", username).Scan(&role)
	return role, err
}

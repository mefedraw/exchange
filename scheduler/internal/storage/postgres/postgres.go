package postgres

import (
	"database/sql"
	"fmt"
)

const (
	host     = "localhost"
	port     = "5432"
	user     = "ted"
	password = "ted"
	dbname   = "scheduler"
)

type Storage struct {
	db *sql.DB
}

func New() (*Storage, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("error connecting to the database: %v", err)
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("error pinging the database: %v", err)
	}
	stmt, err := db.Prepare(
		`CREATE TABLE IF NOT EXISTS users (
    				id SERIAL PRIMARY KEY,
    				mail TEXT NOT NULL,
   					registration_date DATE NOT NULL);
				CREATE INDEX IF NOT EXISTS idx_mail ON users(mail);
	`)
	if err != nil {
		return nil, fmt.Errorf("error creating the users table: %v", err)
	}
	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("error creating the users table: %v", err)
	}
	fmt.Println("Successfully connected!")
	return &Storage{db: db}, nil
}

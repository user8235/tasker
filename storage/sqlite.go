package storage

import (
	"database/sql"
	"log"
	"os"
)

// Используем переменную базы данных
var DB *sql.DB

// Получаем порт из переменной окружения или используем значение по умолчанию
func GetPort() string {
	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540" // Порт по умолчанию
	}
	return port
}

// Получаем путь к базе данных из переменной окружения или используем значение по умолчанию
func GetDBFile() string {
	dbFile := os.Getenv("TODO_DBFILE")
	if dbFile == "" {
		dbFile = "./scheduler.db" // Путь по умолчанию
	}
	return dbFile
}

// Проверяем, существует ли файл
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// Создаём таблицу и индекс
func CreateTable(db *sql.DB) {
	query := `
        CREATE TABLE scheduler (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            date TEXT NOT NULL,
            title TEXT NOT NULL,
            comment TEXT,
            repeat TEXT
        );
        CREATE INDEX idx_date ON scheduler(date);
    `
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("База данных создана")
}

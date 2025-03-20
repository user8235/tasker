package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"tasker/api"
	"tasker/models"
	"tasker/storage"
	"tasker/utils"

	_ "github.com/mattn/go-sqlite3" // Драйвер SQLite
)

//var DB *sql.DB

func main() {
	// Определяем порт
	port := storage.GetPort()

	// Определяем путь к базе данных
	dbFile := storage.GetDBFile()

	// Проверяем, нужно ли создавать базу данных
	if storage.FileExists(dbFile) {
		log.Println("База данных найдена:", dbFile)
	} else {
		log.Println("База данных не найдена, создаём новую:", dbFile)
	}
	install := !storage.FileExists(dbFile)

	// Открываем базу данных
	//var err error
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Инициализируем глобальную переменную базы данных
	models.DB = db

	// Если база данных не существует, создаём таблицу и индекс
	if install {
		storage.CreateTable(db)
	}

	// Настраиваем файловый сервер
	webDir := "./web"
	http.Handle("/", http.FileServer(http.Dir(webDir)))

	// Обработчики API без аутентификации
	http.HandleFunc("/api/nextdate", utils.NextDateHandler)
	http.HandleFunc("/api/tasks", api.TasksHandler)
	http.HandleFunc("/api/task/done", api.DoneTaskHandler)

	// Обработчик для /api/task с поддержкой разных методов HTTP
	http.HandleFunc("/api/task", func(w http.ResponseWriter, r *http.Request) {
		// Устанавливаем заголовки CORS для всех методов
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Если метод OPTIONS, завершаем обработку
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Обрабатываем запросы
		switch r.Method {
		case http.MethodPost:
			api.AddTaskHandler(w, r)
		case http.MethodGet:
			api.GetTaskHandler(w, r)
		case http.MethodPut:
			api.UpdateTaskHandler(w, r)
		case http.MethodDelete:
			api.DeleteTaskHandler(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "Метод не поддерживается"})
		}
	})

	// Запускаем сервер
	log.Printf("Сервер запущен на http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

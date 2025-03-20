package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"tasker/models"
	"time"
)

// Обработчик для получения списка задач
func TasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	// Получаем параметр поиска
	search := r.URL.Query().Get("search")

	// Ссылаемся на глобальную объявленную переменную
	db := models.DB

	var tasks []models.Task

	// Если указан поиск
	if search != "" {
		// Проверяем, является ли поиск датой
		if date, err := time.Parse("02.01.2006", search); err == nil {
			// Поиск по дате
			query := `SELECT * FROM scheduler WHERE date = ? ORDER BY date LIMIT 50`
			rows, err := db.Query(query, date.Format("20060102"))
			if err != nil {
				log.Println("Ошибка при выполнении запроса:", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(models.TasksResponse{Error: "Ошибка при выполнении запроса"})
				return
			}
			defer rows.Close()

			for rows.Next() {
				var task models.Task
				if err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat); err != nil {
					log.Println("Ошибка при сканировании строки:", err)
					continue
				}
				tasks = append(tasks, task)
			}
			// Проверяем, не возникли ли ошибки после цикла
			if err := rows.Err(); err != nil {
				log.Fatalf("Ошибка при чтении данных: %v", err)
			}
		} else {
			// Поиск по заголовку или комментарию
			query := `SELECT * FROM scheduler WHERE title LIKE ? OR comment LIKE ? ORDER BY date LIMIT 50`
			searchPattern := "%" + search + "%"
			rows, err := db.Query(query, searchPattern, searchPattern)
			if err != nil {
				log.Println("Ошибка при выполнении запроса:", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(models.TasksResponse{Error: "Ошибка при выполнении запроса"})
				return
			}
			defer rows.Close()

			for rows.Next() {
				var task models.Task
				if err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat); err != nil {
					log.Println("Ошибка при сканировании строки:", err)
					continue
				}
				tasks = append(tasks, task)
			}
			// Проверяем, не возникли ли ошибки после цикла
			if err := rows.Err(); err != nil {
				log.Fatalf("Ошибка при чтении данных: %v", err)
			}
		}
	} else {
		// Получаем все задачи, отсортированные по дате
		query := `SELECT * FROM scheduler ORDER BY date LIMIT 50`
		rows, err := db.Query(query)
		if err != nil {
			log.Println("Ошибка при выполнении запроса:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.TasksResponse{Error: "Ошибка при выполнении запроса"})
			return
		}
		defer rows.Close()

		for rows.Next() {
			var task models.Task
			if err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat); err != nil {
				log.Println("Ошибка при сканировании строки:", err)
				continue
			}
			tasks = append(tasks, task)
		}
		// Проверяем, не возникли ли ошибки после цикла
		if err := rows.Err(); err != nil {
			log.Fatalf("Ошибка при чтении данных: %v", err)
		}
	}

	// Если задач нет, возвращаем пустой список
	if tasks == nil {
		tasks = []models.Task{}
	}

	// Возвращаем список задач
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.TasksResponse{Tasks: tasks})
}

// Обработчик для получения задачи по ID
func GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	// Получаем идентификатор задачи из параметра запроса
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Не указан идентификатор"})
		return
	}

	// Открываем базу данных
	db, err := sql.Open("sqlite3", "./scheduler.db")
	if err != nil {
		log.Println("Ошибка при открытии базы данных:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка при подключении к базе данных"})
		return
	}
	defer db.Close()

	// Получаем задачу из базы данных
	var task models.Task
	err = db.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id).Scan(
		&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Задача не найдена"})
		} else {
			log.Println("Ошибка при выполнении запроса:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка при выполнении запроса"})
		}
		return
	}

	// Возвращаем задачу
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(task)
}

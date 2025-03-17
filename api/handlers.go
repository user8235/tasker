package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"tasker/models"
	"tasker/utils"
	"time"
)

// Обработчик для получения списка задач
func TasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	// Получаем параметр поиска
	search := r.URL.Query().Get("search")

	// Открываем базу данных
	db, err := sql.Open("sqlite3", "./scheduler.db")
	if err != nil {
		log.Println("Ошибка при открытии базы данных:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.TasksResponse{Error: "Ошибка при подключении к базе данных"})
		return
	}
	defer db.Close()

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

// Обработчик для обновления задачи
func UpdateTaskHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	// Парсим JSON-запрос
	var task models.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка десериализации JSON"})
		return
	}

	// Проверяем обязательное поле "id"
	if task.ID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Не указан идентификатор задачи"})
		return
	}

	// Проверяем корректность и формат даты
	_, err := time.Parse("20060102", task.Date)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Не верный формат даты"})
		return
	}

	// Проверяем обязательное поле "Title"
	if task.Title == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Отсутствует заголовок задачи"})
		return
	}

	// Проверяем формат поля repeat
	isValidRepeat := func(repeat string) bool {
		if repeat == "" {
			return true
		}
		parts := strings.Fields(repeat)
		if len(parts) != 2 || parts[0] != "d" {
			return false
		}
		_, err := strconv.Atoi(parts[1])
		return err == nil
	}

	if !isValidRepeat(task.Repeat) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Некорректный формат repeat"})
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

	// Обновляем задачу в базе данных
	query := `UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat = ? WHERE id = ?`
	result, err := db.Exec(query, task.Date, task.Title, task.Comment, task.Repeat, task.ID)
	if err != nil {
		log.Println("Ошибка при выполнении запроса:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка при обновлении задачи"})
		return
	}

	// Проверяем, была ли обновлена задача
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println("Ошибка при проверке обновления:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка при проверке обновления"})
		return
	}
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Задача не найдена"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{})
}

// Обработчик для отметки задачи как выполненной
func DoneTaskHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	// Получаем идентификатор задачи из параметра запроса
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Не указан идентификатор задачи"})
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

	// Если задача периодическая, обновляем дату
	if task.Repeat != "" {
		now := time.Now()
		nextDate, err := utils.NextDate(now, task.Date, task.Repeat)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка при расчете следующей даты"})
			return
		}

		// Обновляем дату задачи в базе данных
		_, err = db.Exec("UPDATE scheduler SET date = ? WHERE id = ?", nextDate, id)
		if err != nil {
			log.Println("Ошибка при обновлении задачи:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка при обновлении задачи"})
			return
		}
	} else {
		// Если задача одноразовая, удаляем её
		_, err = db.Exec("DELETE FROM scheduler WHERE id = ?", id)
		if err != nil {
			log.Println("Ошибка при удалении задачи:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка при удалении задачи"})
			return
		}
	}

	// Возвращаем успешный ответ
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{})
}

// Обработчик для удаления задачи
func DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	// Получаем идентификатор задачи из параметра запроса
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Не указан идентификатор задачи"})
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

	// Удаляем задачу из базы данных
	result, err := db.Exec("DELETE FROM scheduler WHERE id = ?", id)
	if err != nil {
		log.Println("Ошибка при удалении задачи:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка при удалении задачи"})
		return
	}

	// Проверяем, была ли задача удалена
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println("Ошибка при проверке удаления:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка при проверке удаления"})
		return
	}
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Задача не найдена"})
		return
	}

	// Возвращаем успешный ответ
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{})
}

// Обработчик для добавления задачи
func AddTaskHandler(w http.ResponseWriter, r *http.Request) {
	// Устанавливаем заголовок Content-Type
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	// Проверяем метод запроса
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.TaskResponse{Error: "Метод не поддерживается"})
		return
	}

	// Парсим JSON-запрос
	var task models.TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.TaskResponse{Error: "Ошибка десериализации JSON"})
		return
	}

	// Проверяем обязательное поле "title"
	if task.Title == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.TaskResponse{Error: "Не указан заголовок задачи"})
		return
	}

	// Парсим дату задачи
	var taskDate time.Time
	if task.Date == "" || task.Date == "today" || task.Date == time.Now().Format("20060102") {
		taskDate = time.Now() // Используем текущую дату, если дата не указана или равна "today"
	} else {
		var err error
		taskDate, err = time.Parse("20060102", task.Date)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.TaskResponse{Error: "Неверный формат даты"})
			return
		}
	}

	// Если дата задачи меньше текущей, вычисляем следующую дату
	now := time.Now()
	if taskDate.Before(now) {
		if task.Repeat == "" {
			taskDate = now // Если правило повторения не указано, используем текущую дату
		} else {
			nextDate, err := utils.NextDate(now, taskDate.Format("20060102"), task.Repeat)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(models.TaskResponse{Error: err.Error()})
				return
			}
			taskDate, _ = time.Parse("20060102", nextDate)
		}
	}

	// Проверяем правило повторения
	if task.Repeat != "" {
		_, err := utils.NextDate(now, taskDate.Format("20060102"), task.Repeat)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.TaskResponse{Error: "Неверный формат правила повторения"})
			return
		}
	}

	// Добавляем задачу в базу данных
	db, err := sql.Open("sqlite3", "./scheduler.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	query := `INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)`
	res, err := db.Exec(query, taskDate.Format("20060102"), task.Title, task.Comment, task.Repeat)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.TaskResponse{Error: "Ошибка при добавлении задачи"})
		return
	}

	// Получаем ID добавленной задачи
	id, err := res.LastInsertId()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.TaskResponse{Error: "Ошибка при получении ID задачи"})
		return
	}

	// Возвращаем успешный ответ
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.TaskResponse{ID: id})
}

package api

import (
	"encoding/json"
	"net/http"
	"tasker/models"
	"tasker/utils"
	"time"
)

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

	// Ссылаемся на глобальную объявленную переменную
	db := models.DB

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

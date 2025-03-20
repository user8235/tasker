package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"tasker/models"
	"time"
)

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

	// Ссылаемся на глобальную объявленную переменную
	db := models.DB

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

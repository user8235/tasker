package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"tasker/models"
	"tasker/utils"
	"time"
)

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

	// Ссылаемся на глобальную объявленную переменную
	db := models.DB

	// Получаем задачу из базы данных
	var task models.Task
	err := db.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id).Scan(
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

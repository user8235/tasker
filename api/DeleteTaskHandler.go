package api

import (
	"encoding/json"
	"log"
	"net/http"
	"tasker/models"
)

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

	// Ссылаемся на глобальную объявленную переменную
	db := models.DB

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

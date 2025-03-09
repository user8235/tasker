package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // Драйвер SQLite
)

// Структура для парсинга JSON-запроса
type TaskRequest struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

// Структура для JSON-ответа
type TaskResponse struct {
	ID    int64  `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

// Получаем порт из переменной окружения или используем значение по умолчанию
func getPort() string {
	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540" // Порт по умолчанию
	}
	return port
}

// Получаем путь к базе данных из переменной окружения или используем значение по умолчанию
func getDBFile() string {
	dbFile := os.Getenv("TODO_DBFILE")
	if dbFile == "" {
		dbFile = "./scheduler.db" // Путь по умолчанию
	}
	return dbFile
}

// Проверяем, существует ли файл
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// Создаём таблицу и индекс
func createTable(db *sql.DB) {
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

// isLeapYear проверяет, является ли год високосным
func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// NextDate вычисляет следующую дату выполнения задачи
func NextDate(now time.Time, date string, repeat string) (string, error) {
	// Парсим исходную дату
	taskDate, err := time.Parse("20060102", date)
	if err != nil {
		return "", errors.New("неверный формат даты")
	}

	// Если правило пустое, возвращаем ошибку
	if repeat == "" {
		return "", errors.New("правило повторения не указано")
	}

	// Обрабатываем правила
	switch {
	case strings.HasPrefix(repeat, "d "): // Правило "d <число>"
		days, err := strconv.Atoi(repeat[2:])
		if err != nil || days <= 0 || days > 400 {
			return "", errors.New("неверный формат правила d")
		}
		for {
			taskDate = taskDate.AddDate(0, 0, days) // Добавляем дни
			if taskDate.After(now) || taskDate.Equal(now) {
				return taskDate.Format("20060102"), nil
			}
		}

	case repeat == "y": // Правило "y"
		for {
			taskDate = taskDate.AddDate(1, 0, 0) // Добавляем год
			// Проверяем, существует ли такая дата в следующем году
			if taskDate.Month() == time.February && taskDate.Day() == 29 {
				if !isLeapYear(taskDate.Year()) {
					taskDate = time.Date(taskDate.Year(), 3, 1, 0, 0, 0, 0, time.UTC)
				}
			}
			if taskDate.After(now) || taskDate.Equal(now) {
				return taskDate.Format("20060102"), nil
			}
		}

	case strings.HasPrefix(repeat, "w "): // Правило "w <дни недели>"
		daysOfWeek := strings.Split(repeat[2:], ",")
		validDays := make(map[int]bool)
		for _, dayStr := range daysOfWeek {
			day, err := strconv.Atoi(strings.TrimSpace(dayStr))
			if err != nil || day < 1 || day > 7 {
				return "", errors.New("неверный формат правила w")
			}
			validDays[day] = true
		}

		for {
			taskDate = taskDate.AddDate(0, 0, 1)                           // Переходим к следующему дню
			if validDays[int(taskDate.Weekday())] && taskDate.After(now) { // Проверяем, что дата строго больше now
				return taskDate.Format("20060102"), nil
			}
		}

	case strings.HasPrefix(repeat, "m "): // Правило "m <дни месяца> [месяцы]"
		parts := strings.Split(repeat[2:], " ")
		if len(parts) == 0 || len(parts) > 2 {
			return "", errors.New("неверный формат правила m")
		}

		// Парсим дни месяца
		days := strings.Split(parts[0], ",")
		validDays := make(map[int]bool)
		for _, dayStr := range days {
			day, err := strconv.Atoi(strings.TrimSpace(dayStr))
			if err != nil || day < -2 || day == 0 || day > 31 {
				return "", errors.New("неверный формат правила m: дни")
			}
			validDays[day] = true
		}

		// Парсим месяцы (если указаны)
		validMonths := make(map[int]bool)
		if len(parts) == 2 {
			months := strings.Split(parts[1], ",")
			for _, monthStr := range months {
				month, err := strconv.Atoi(strings.TrimSpace(monthStr))
				if err != nil || month < 1 || month > 12 {
					return "", errors.New("неверный формат правила m: месяцы")
				}
				validMonths[month] = true
			}
		}

		for {
			taskDate = taskDate.AddDate(0, 0, 1)
			if (len(validMonths) == 0 || validMonths[int(taskDate.Month())]) && isValidDay(taskDate, validDays) && (taskDate.After(now) || taskDate.Equal(now)) {
				return taskDate.Format("20060102"), nil
			}
		}

	default:
		return "", errors.New("неподдерживаемый формат правила")
	}
}

// isValidDay проверяет, соответствует ли день месяца правилу
func isValidDay(date time.Time, validDays map[int]bool) bool {
	day := date.Day()
	if validDays[day] {
		return true
	}
	if validDays[-1] && date.Day() == lastDayOfMonth(date) {
		return true
	}
	if validDays[-2] && date.Day() == lastDayOfMonth(date)-1 {
		return true
	}
	return false
}

// lastDayOfMonth возвращает последний день месяца
func lastDayOfMonth(date time.Time) int {
	return time.Date(date.Year(), date.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func nextDateHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем параметры из запроса
	nowStr := r.FormValue("now")
	dateStr := r.FormValue("date")
	repeat := r.FormValue("repeat")

	// Парсим текущую дату
	now, err := time.Parse("20060102", nowStr)
	if err != nil {
		http.Error(w, "неверный формат параметра now", http.StatusBadRequest)
		return
	}

	// Вычисляем следующую дату
	nextDate, err := NextDate(now, dateStr, repeat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Возвращаем результат
	fmt.Fprintf(w, nextDate)
}

// Обработчик для добавления задачи
func addTaskHandler(w http.ResponseWriter, r *http.Request) {
	// Устанавливаем заголовок Content-Type
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	// Проверяем метод запроса
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(TaskResponse{Error: "Метод не поддерживается"})
		return
	}

	// Парсим JSON-запрос
	var task TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TaskResponse{Error: "Ошибка десериализации JSON"})
		return
	}

	// Проверяем обязательное поле "title"
	if task.Title == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TaskResponse{Error: "Не указан заголовок задачи"})
		return
	}

	// Парсим дату задачи
	var taskDate time.Time
	if task.Date == "" || task.Date == "today" {
		taskDate = time.Now() // Используем текущую дату, если дата не указана или равна "today"
	} else {
		var err error
		taskDate, err = time.Parse("20060102", task.Date)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(TaskResponse{Error: "Неверный формат даты"})
			return
		}
	}

	// Если дата задачи меньше текущей, вычисляем следующую дату
	now := time.Now()
	if taskDate.Before(now) {
		if task.Repeat == "" {
			taskDate = now // Если правило повторения не указано, используем текущую дату
		} else {
			nextDate, err := NextDate(now, taskDate.Format("20060102"), task.Repeat)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(TaskResponse{Error: err.Error()})
				return
			}
			taskDate, _ = time.Parse("20060102", nextDate)
		}
	}

	// Проверяем правило повторения
	if task.Repeat != "" {
		_, err := NextDate(now, taskDate.Format("20060102"), task.Repeat)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(TaskResponse{Error: "Неверный формат правила повторения"})
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
		json.NewEncoder(w).Encode(TaskResponse{Error: "Ошибка при добавлении задачи"})
		return
	}

	// Получаем ID добавленной задачи
	id, err := res.LastInsertId()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(TaskResponse{Error: "Ошибка при получении ID задачи"})
		return
	}

	// Возвращаем успешный ответ
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(TaskResponse{ID: id})
}

func main() {
	// Определяем порт
	port := getPort()

	// Определяем путь к базе данных
	dbFile := getDBFile()

	// Проверяем, нужно ли создавать базу данных
	install := !fileExists(dbFile)

	// Открываем базу данных
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Если база данных не существует, создаём таблицу и индекс
	if install {
		createTable(db)
	}

	// Настраиваем файловый сервер
	webDir := "./web"
	http.Handle("/", http.FileServer(http.Dir(webDir)))
	http.HandleFunc("/api/nextdate", nextDateHandler)
	http.HandleFunc("/api/task", addTaskHandler)

	// Запускаем сервер
	log.Printf("Сервер запущен на http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

package utils

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

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

		// Если текущая дата уже является датой выполнения, возвращаем её
		if taskDate.Equal(now) {
			return taskDate.Format("20060102"), nil
		}

		// Иначе добавляем дни, пока не получим дату, которая больше или равна now
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

func NextDateHandler(w http.ResponseWriter, r *http.Request) {
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

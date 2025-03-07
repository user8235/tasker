package main

import (
	"log"
	"net/http"
	"os"
)

// Получаем порт из переменной окружения или используем значение по умолчанию
func getPort() string {
	port := os.Getenv("TASKER_PORT")
	if port == "" {
		port = "7540" // Порт по умолчанию
	}
	return port
}

func main() {
	// Определяем порт
	port := getPort()

	// Указываем директорию с файлами фронтенда
	webDir := "./web"

	// Настраиваем файловый сервер
	http.Handle("/", http.FileServer(http.Dir(webDir)))

	// Запускаем сервер
	log.Printf("Сервер запущен на http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

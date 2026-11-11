package main

import (
	"fmt"
	"net/http"

	"github.com/tiger-box/URL-Shortener/internal"
	"github.com/tiger-box/URL-Shortener/internal/handler"
)

func main() {
	shortener := internal.NewURLShortener()
	urlHandler := handler.NewURL(shortener)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.Health)
	mux.HandleFunc("POST /shorten", urlHandler.GetShorter)
	mux.HandleFunc("GET /{code}", urlHandler.GetURL)

	// Запускаем сервер на порту 8080
	fmt.Println("Starting server at port 8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		fmt.Println("Error starting the server:", err)
	}
}

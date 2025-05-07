package main

import (
	"log"

	"github.com/jaxxiy/newforum/forum_service/internal/app"
)

func main() {
	srv := app.NewServer()
	if err := srv.Run(); err != nil {
		log.Fatalf("Ошибка запуска: %v", err)
	}
}

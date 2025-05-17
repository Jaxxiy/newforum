package main

import (
	"database/sql"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jaxxiy/newforum/core/logger"
	_ "github.com/jaxxiy/newforum/forum_service/docs"
	"github.com/jaxxiy/newforum/forum_service/internal/app"
	"github.com/jaxxiy/newforum/forum_service/internal/handlers"
	"github.com/jaxxiy/newforum/forum_service/internal/repository"
	_ "github.com/lib/pq"
	httpSwagger "github.com/swaggo/http-swagger"
)

var log = logger.GetLogger()

// @title Forum Service API
// @version 1.0
// @description Forum Service API for managing forums and messages
// @host localhost:8080
// @BasePath /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// Set development mode for more detailed logging
	logger.SetDevelopmentMode(true)

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:Stas2005101010!@localhost:5432/forum?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database", logger.Error(err))
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database", logger.Error(err))
	}

	forumsRepo := repository.NewForumsRepo(db)

	r := mux.NewRouter()

	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	handlers.RegisterForumHandlers(r, forumsRepo)

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	server, err := app.NewServer(httpPort)
	if err != nil {
		log.Fatal("Failed to create server", logger.Error(err))
	}

	log.Info("Starting HTTP server", logger.String("port", httpPort))

	if err := server.Run(); err != nil {
		log.Fatal("Failed to start HTTP server", logger.Error(err))
	}

	log.Info("Server stopped gracefully")
}

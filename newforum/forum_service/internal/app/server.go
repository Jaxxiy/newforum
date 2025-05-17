package app

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/gorilla/mux"
	"github.com/jaxxiy/newforum/core/logger"
	"github.com/jaxxiy/newforum/forum_service/internal/handlers"
	"github.com/jaxxiy/newforum/forum_service/internal/repository"
	"google.golang.org/grpc"
)

var log = logger.GetLogger()

type Server struct {
	httpServer *http.Server
	grpcServer *grpc.Server
	db         *sql.DB
}

func NewServer(port string) (*Server, error) {
	db, err := sql.Open("postgres", os.Getenv("DB_DSN"))
	if err != nil {
		log.Fatal("Failed to connect to database", logger.Error(err))
		return nil, err
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal("Failed to create migrations driver", logger.Error(err))
		return nil, err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		log.Fatal("Failed to initialize migrations", logger.Error(err))
		return nil, err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("Failed to apply migrations", logger.Error(err))
		return nil, err
	}

	repo := repository.NewForumsRepo(db)
	router := mux.NewRouter()

	handlers.RegisterForumHandlers(router, repo)

	return &Server{
		httpServer: &http.Server{
			Addr:    ":" + port,
			Handler: router,
		},
		db: db,
	}, nil
}

func (s *Server) Run() error {
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil {
			log.Error("HTTP server error", logger.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Info("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}

	return s.db.Close()
}

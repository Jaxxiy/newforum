package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	//"github.com/golang-migrate/migrate/v4"
	//"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/gorilla/mux"
	"github.com/jaxxiy/newforum/forum_service/internal/handlers"
	"github.com/jaxxiy/newforum/forum_service/internal/repository"
	"github.com/jaxxiy/newforum/forum_service/internal/service"
	"google.golang.org/grpc"
)

type Server struct {
	httpServer *http.Server
	grpcServer *grpc.Server
	db         *repository.Postgres
	wg         sync.WaitGroup
}

func NewServer() *Server {
	r := mux.NewRouter()

	dsn := "postgres://postgres:Stas2005101010!@localhost:5432/forum?sslmode=disable"

	db, err := repository.NewPostgres(dsn)
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}

	//driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	//if err != nil {
	//	log.Fatalf("Не удалось создать драйвер миграций: %v", err)
	//}

	//m, err := migrate.NewWithDatabaseInstance(
	//	"file://../../migrations",
	//	"forum",
	//	driver,
	//)

	if err != nil {
		log.Fatalf("Ошибка инициализации миграций: %v", err)
	}

	// Применяем миграции
	//if err := m.Up(); err != nil && err != migrate.ErrNoChange {
	//	log.Fatalf("Ошибка применения миграций: %v", err)
	//}

	forumRepo := repository.NewForumsRepo(db.DB)

	handlers.RegisterForumHandlers(r, forumRepo)

	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("C:/Users/Soulless/Desktop/myforum/cmd/frontend/"))))

	httpSrv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go service.StartWebSocket()

	return &Server{
		httpServer: httpSrv,
		db:         db,
	}
}

func (s *Server) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP остановился: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Завершение работы...")
	if err := s.httpServer.Shutdown(context.Background()); err != nil {
		return err
	}
	s.wg.Wait()
	return nil
}

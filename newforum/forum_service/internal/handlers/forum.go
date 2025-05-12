package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/jaxxiy/newforum/core/pkg/jwt"
	"github.com/jaxxiy/newforum/forum_service/internal/grpc"
	"github.com/jaxxiy/newforum/forum_service/internal/models"
	"github.com/jaxxiy/newforum/forum_service/internal/repository"
)

var (
	templates = template.Must(template.ParseGlob("C:/Users/Soulless/Desktop/newforum/core/templates/*.html"))
	upgrader  = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Для разработки
		},
	}
	clients   = make(map[int]map[*websocket.Conn]bool) // forumID -> connections
	clientsMu sync.RWMutex

	//mini-chat
	globalChatClients  = make(map[*websocket.Conn]bool)
	globalChatMu       sync.RWMutex
	globalChatUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Для разработки
		},
	}
	globalChatBroadcast = make(chan GlobalChatMessage)
	globalChatHistory   []GlobalChatMessage

	authClient grpc.AuthClient
)

func init() {
	var err error
	authClient, err = grpc.NewClient("localhost:50051")
	if err != nil {
		log.Fatalf("Failed to create auth client: %v", err)
	}
}

type GlobalChatMessageRequest struct {
	Author  string `json:"username"`
	Content string `json:"text"`
}

type GlobalChatMessage struct {
	Author    string    `json:"username"`
	Content   string    `json:"text"`
	CreatedAt time.Time `json:"timestamp"`
}

type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func RegisterForumHandlers(r *mux.Router, repo repository.ForumsRepository) {

	r.HandleFunc("/ws/global", func(w http.ResponseWriter, r *http.Request) {
		serveGlobalChat(w, r, repo)
	})

	r.HandleFunc("/ws/{forum_id:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
		serveWebSocket(w, r)
	})
	go handleGlobalChatMessages()

	api := r.PathPrefix("/api").Subrouter()

	// Auth page routes
	r.HandleFunc("/auth/login", LoginPage).Methods("GET")
	r.HandleFunc("/auth/register", RegisterPage).Methods("GET")

	api.HandleFunc("/forums", ListForums(repo)).Methods("GET")
	api.HandleFunc("/forums/new", NewForumForm()).Methods("GET")
	api.HandleFunc("/forums", CreateForum(repo)).Methods("POST")
	api.HandleFunc("/forums/{id:[0-9]+}", GetForum(repo)).Methods("GET")
	api.HandleFunc("/forums/{id:[0-9]+}", UpdateForum(repo)).Methods("PUT")
	api.HandleFunc("/forums/{id:[0-9]+}", DeleteForum(repo)).Methods("DELETE")

	// Обработчики сообщений
	api.HandleFunc("/forums/{id:[0-9]+}/messages", GetMessages(repo)).Methods("GET")
	api.HandleFunc("/forums/{id:[0-9]+}/messages", PostMessage(repo)).Methods("POST")
	api.HandleFunc("/forums/{forum_id:[0-9]+}/messages/{message_id:[0-9]+}", DeleteMessage(repo)).Methods("DELETE")
	api.HandleFunc("/forums/{id:[0-9]+}/messages/{message_id:[0-9]+}", UpdateMessage(repo)).Methods("PUT")

	api.HandleFunc("/global-chat", handleGlobalChatMessage(repo)).Methods("POST")

	// Новый API-эндпоинт для загрузки сообщений с учетом токена
	api.HandleFunc("/forums/{id:[0-9]+}/messages-list", GetMessagesAPI(repo)).Methods("GET")

}

func LoginPage(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "login.html", nil)
}

func RegisterPage(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "register.html", nil)
}

// Улучшенный обработчик WebSocket
func serveWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	forumID, err := strconv.Atoi(vars["forum_id"])
	if err != nil {
		http.Error(w, "Invalid forum ID", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer func() {
		unregisterClient(forumID, conn)
		conn.Close()
	}()

	registerClient(forumID, conn)

	// Настройка keep-alive
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Чтение сообщений (для поддержания соединения)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

// Отправка сообщения всем клиентам форума
func broadcastToForum(forumID int, message WSMessage) {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	if conns, ok := clients[forumID]; ok {
		for conn := range conns {
			if err := conn.WriteJSON(message); err != nil {
				log.Printf("WS send error: %v", err)
				go handleFailedConnection(forumID, conn)
			}
		}
	}
}

func handleFailedConnection(forumID int, conn *websocket.Conn) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	if conns, ok := clients[forumID]; ok {
		conn.Close()
		delete(conns, conn)
		log.Printf("Connection removed for forum %d", forumID)
	}
}

func registerClient(forumID int, conn *websocket.Conn) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	if clients[forumID] == nil {
		clients[forumID] = make(map[*websocket.Conn]bool)
	}
	clients[forumID][conn] = true
	log.Printf("New client for forum %d. Total: %d", forumID, len(clients[forumID]))
}

func unregisterClient(forumID int, conn *websocket.Conn) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	if clients[forumID] != nil {
		delete(clients[forumID], conn)
	}
}

// Улучшенный обработчик сообщений
func PostMessage(repo repository.ForumsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Получаем forumID из URL
		vars := mux.Vars(r)
		forumID, err := strconv.Atoi(vars["id"])
		if err != nil {
			sendError(w, http.StatusBadRequest, "Invalid forum ID")
			return
		}

		// Проверяем Content-Type
		if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			sendError(w, http.StatusBadRequest, "Content-Type must be application/json")
			return
		}

		// Декодируем JSON
		var req struct {
			Author  string `json:"author"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid JSON format")
			return
		}

		// Валидация
		if strings.TrimSpace(req.Author) == "" || strings.TrimSpace(req.Content) == "" {
			sendError(w, http.StatusBadRequest, "Author and content are required")
			return
		}

		// Получаем пользователя из токена
		authHeader := r.Header.Get("Authorization")
		var user *models.User
		if authHeader != "" {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if claims, err := jwt.ParseToken(tokenString, "your-secret-key"); err == nil {
				if u, err := repo.GetUserByID(claims.UserID); err == nil {
					user = u
				}
			}
		}
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Проверяем права: автор или admin
		if user.Username != req.Author && user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Создаем сообщение
		msg := models.Message{
			ForumID:   forumID,
			Author:    req.Author,
			Content:   req.Content,
			CreatedAt: time.Now(),
		}

		fmt.Println(msg.CreatedAt)

		// Сохраняем в БД
		id, err := repo.CreateMessage(msg)
		if err != nil {
			log.Printf("DB error: %v", err)
			sendError(w, http.StatusInternalServerError, "Failed to save message")
			return
		}
		msg.ID = id

		// Отправляем через WebSocket
		go broadcastToForum(forumID, WSMessage{
			Type:    "message_created",
			Payload: msg,
		})

		// Успешный ответ
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(msg)
	}
}

func sendError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// sendWSMessage отправляет сообщение всем клиентам в указанном форуме
func sendWSMessage(forumID int, message WSMessage) {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	if conns, ok := clients[forumID]; ok {
		for conn := range conns {
			if err := conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket send error: %v", err)
				go handleFailedConnection(forumID, conn)
			}
		}
	}
}

// ListForums godoc
// @Summary Get all forums
// @Description Get a list of all forums
// @Tags forums
// @Produce json
// @Success 200 {array} models.Forum
// @Router /forums [get]
func ListForums(repo repository.ForumsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		forums, err := repo.GetAll()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		renderTemplate(w, "list_forums.html", map[string]interface{}{
			"Forums": forums,
		})
	}
}

func NewForumForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, "new_forum.html", nil)
	}
}

// CreateForum godoc
// @Summary Create a new forum
// @Description Create a new forum with title and description
// @Tags forums
// @Accept json
// @Produce json
// @Param forum body models.Forum true "Forum info"
// @Success 201 {object} models.Forum
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /forums [post]
func CreateForum(repo repository.ForumsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		title := r.FormValue("title")
		description := r.FormValue("description")

		forum := models.Forum{
			Title:       title,
			Description: description,
			CreatedAt:   time.Now(),
		}

		id, err := repo.Create(forum)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		forum.ID = id

		// Отправляем уведомление через WebSocket
		sendWSMessage(id, WSMessage{
			Type: "forum_created",
			Payload: map[string]interface{}{
				"forum": forum,
			},
		})

		http.Redirect(w, r, "/api/forums", http.StatusSeeOther)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(forum)

	}
}

// GetForum godoc
// @Summary Get forum by ID
// @Description Get forum details by ID
// @Tags forums
// @Produce json
// @Param id path int true "Forum ID"
// @Success 200 {object} models.Forum
// @Failure 404 {object} map[string]string
// @Router /forums/{id} [get]
func GetForum(repo repository.ForumsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Некорректный ID", http.StatusBadRequest)
			return
		}
		f, err := repo.GetByID(id)
		if err != nil {
			http.Error(w, "Форум не найден", http.StatusNotFound)
			return
		}

		renderTemplate(w, "forum_detail.html", f)
	}
}

// GetAllForums возвращает все форумы
func GetAllForums(repo repository.ForumsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		forums, err := repo.GetAll()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(forums)
	}
}

// UpdateForum godoc
// @Summary Update forum
// @Description Update forum details
// @Tags forums
// @Accept json
// @Produce json
// @Param id path int true "Forum ID"
// @Param forum body models.Forum true "Forum info"
// @Success 200 {object} models.Forum
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /forums/{id} [put]
func UpdateForum(repo repository.ForumsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.Atoi(vars["id"])

		var forum models.Forum
		if err := json.NewDecoder(r.Body).Decode(&forum); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := repo.Update(id, forum); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// DeleteForum godoc
// @Summary Delete forum
// @Description Delete forum by ID
// @Tags forums
// @Param id path int true "Forum ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Router /forums/{id} [delete]
func DeleteForum(repo repository.ForumsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.Atoi(vars["id"])

		if err := repo.Delete(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// GetMessages godoc
// @Summary Get forum messages
// @Description Get all messages for a specific forum
// @Tags messages
// @Produce json
// @Param id path int true "Forum ID"
// @Success 200 {array} models.Message
// @Failure 404 {object} map[string]string
// @Router /forums/{id}/messages [get]
func GetMessages(repo repository.ForumsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		forumID, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Invalid forum ID", http.StatusBadRequest)
			return
		}

		// Получаем форум
		forum, err := repo.GetByID(forumID)
		if err != nil {
			http.Error(w, "Forum not found", http.StatusNotFound)
			return
		}

		// Получаем сообщения
		messages, err := repo.GetMessages(forumID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Получаем текущего пользователя и роль из JWT токена
		authHeader := r.Header.Get("Authorization")
		var currentUser, currentRole string
		if authHeader != "" {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if claims, err := jwt.ParseToken(tokenString, "your-secret-key"); err == nil {
				if user, err := repo.GetUserByID(claims.UserID); err == nil {
					currentUser = user.Username
					currentRole = user.Role
				}
			}
		}

		// Рендерим шаблон (если нужно использовать роль в шаблоне)
		data := struct {
			Forum       *models.Forum
			Messages    []models.Message
			CurrentUser string
			CurrentRole string
		}{
			Forum:       forum,
			Messages:    messages,
			CurrentUser: currentUser,
			CurrentRole: currentRole,
		}

		renderTemplate(w, "message_list.html", data)
	}
}

// UpdateMessage godoc
// @Summary Update message
// @Description Update an existing message
// @Tags messages
// @Accept json
// @Produce json
// @Param forum_id path int true "Forum ID"
// @Param message_id path int true "Message ID"
// @Param message body models.Message true "Message info"
// @Success 200 {object} models.Message
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /forums/{forum_id}/messages/{message_id} [put]
func UpdateMessage(repo repository.ForumsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем ID сообщения из URL
		vars := mux.Vars(r)
		messageID, err := strconv.Atoi(vars["message_id"])
		fmt.Println(messageID)
		if err != nil {
			http.Error(w, "Invalid message ID", http.StatusBadRequest)
			return
		}

		// Парсим тело запроса
		var request struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Получаем пользователя из токена
		authHeader := r.Header.Get("Authorization")
		var user *models.User
		if authHeader != "" {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if claims, err := jwt.ParseToken(tokenString, "your-secret-key"); err == nil {
				if u, err := repo.GetUserByID(claims.UserID); err == nil {
					user = u
				}
			}
		}
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Получаем сообщение
		msg, err := repo.GetMessageByID(messageID)
		if err != nil {
			http.Error(w, "Message not found", http.StatusNotFound)
			return
		}

		// Проверяем права: автор или admin
		if user.Username != msg.Author && user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Обновляем сообщение в репозитории
		updatedMessage, err := repo.PutMessage(messageID, request.Content)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Возвращаем обновленное сообщение
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updatedMessage)
	}
}

// Отправка сообщения

// DeleteMessage godoc
// @Summary Delete message
// @Description Delete a message from a forum
// @Tags messages
// @Param forum_id path int true "Forum ID"
// @Param message_id path int true "Message ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Router /forums/{forum_id}/messages/{message_id} [delete]
func DeleteMessage(repo repository.ForumsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		messageID, err := strconv.Atoi(vars["message_id"])
		if err != nil {
			http.Error(w, "Invalid message ID", http.StatusBadRequest)
			return
		}

		// Получаем пользователя из токена
		authHeader := r.Header.Get("Authorization")
		var user *models.User
		if authHeader != "" {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if claims, err := jwt.ParseToken(tokenString, "your-secret-key"); err == nil {
				if u, err := repo.GetUserByID(claims.UserID); err == nil {
					user = u
				}
			}
		}
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Получаем сообщение
		msg, err := repo.GetMessageByID(messageID)
		if err != nil {
			http.Error(w, "Message not found", http.StatusNotFound)
			return
		}

		// Проверяем права: автор или admin
		if user.Username != msg.Author && user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		err = repo.DeleteMessage(messageID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func serveGlobalChat(w http.ResponseWriter, r *http.Request, repo repository.ForumsRepository) {
	conn, err := globalChatUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Global chat WebSocket upgrade error: %v", err)
		return
	}

	log.Println("Новое WebSocket соединение установлено")

	defer func() {
		log.Println("WebSocket соединение закрыто")
		globalChatMu.Lock()
		delete(globalChatClients, conn)
		globalChatMu.Unlock()
		conn.Close()
	}()

	// Регистрация клиента
	globalChatMu.Lock()
	globalChatClients[conn] = true
	globalChatMu.Unlock()

	// Загрузка истории из БД (последние 100 сообщений)
	history, err := repo.GetGlobalChatHistory(100)
	if err != nil {
		log.Printf("Ошибка загрузки истории чата: %v", err)
	} else {
		// Конвертируем в GlobalChatMessage и отправляем
		for _, msg := range history {
			chatMsg := GlobalChatMessage{
				Author:    msg.Author,
				Content:   msg.Content,
				CreatedAt: msg.CreatedAt,
			}
			if err := conn.WriteJSON(chatMsg); err != nil {
				log.Printf("Ошибка отправки истории: %v", err)
				break
			}
		}
	}

	// Чтение новых сообщений
	for {
		var msg GlobalChatMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("Global chat error: %v", err)
			}
			break
		}

		// Сохраняем в БД через репозиторий
		_, err := repo.CreateGlobalMessage(models.GlobalMessage{
			Author:    msg.Author,
			Content:   msg.Content,
			CreatedAt: time.Now(),
		})
		if err != nil {
			log.Printf("Ошибка сохранения сообщения: %v", err)
		}

		// Рассылка всем клиентам
		globalChatBroadcast <- msg
	}
}

func handleGlobalChatMessages() {
	for {
		msg := <-globalChatBroadcast
		globalChatMu.Lock()

		// Обновляем историю в памяти (опционально)
		globalChatHistory = append(globalChatHistory, msg)
		if len(globalChatHistory) > 100 {
			globalChatHistory = globalChatHistory[1:]
		}

		// Рассылка
		for client := range globalChatClients {
			if err := client.WriteJSON(msg); err != nil {
				log.Printf("Ошибка отправки: %v", err)
				client.Close()
				delete(globalChatClients, client)
			}
		}
		globalChatMu.Unlock()
	}
}

// Обработчик POST-запроса для глобального чата
func handleGlobalChatMessage(repo repository.ForumsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// 1. Проверяем Content-Type
		if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, `{"error": "Content-Type must be application/json"}`, http.StatusBadRequest)
			return
		}

		// 2. Парсим JSON
		var req GlobalChatMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "Invalid JSON format"}`, http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// 3. Валидация
		if strings.TrimSpace(req.Author) == "" || strings.TrimSpace(req.Content) == "" {
			http.Error(w, `{"error": "Username and text are required"}`, http.StatusBadRequest)
			return
		}

		// 4. Создаем структуру models.GlobalMessage для сохранения в БД
		msgmodels := models.GlobalMessage{
			Author:    req.Author,
			Content:   req.Content,
			CreatedAt: time.Now(),
		}

		// 5. Сохраняем в БД
		id, err := repo.CreateGlobalMessage(msgmodels)
		if err != nil {
			log.Printf("DB error: %v", err)
			http.Error(w, `{"error": "Failed to save message"}`, http.StatusInternalServerError)
			return
		}

		// 6. Создаем структуру GlobalChatMessage для отправки в WebSocket
		msgWebSocket := GlobalChatMessage{
			Author:    req.Author,
			Content:   req.Content,
			CreatedAt: time.Now(),
		}
		log.Printf("Sending message %v to websocket", msgWebSocket)

		// 7. Отправляем в WebSocket
		globalChatBroadcast <- msgWebSocket

		// 8. Успешный ответ
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        id,
			"username":  req.Author,
			"text":      req.Content,
			"timestamp": time.Now(),
		})

	}
}

// GetMessagesAPI godoc
// @Summary Get forum messages with user info
// @Description Get all messages for a forum with current user info
// @Tags messages
// @Produce json
// @Param id path int true "Forum ID"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /forums/{id}/messages-list [get]
func GetMessagesAPI(repo repository.ForumsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		forumID, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Invalid forum ID", http.StatusBadRequest)
			return
		}

		messages, err := repo.GetMessages(forumID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Безопасная проверка токена
		var currentUser, currentRole string
		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString != "" && authClient != nil {
				user, err := authClient.GetUserByToken(ctx, tokenString)
				if err != nil {
					log.Printf("Error getting user by token: %v", err)
					// Продолжаем выполнение без информации о пользователе
				} else if user != nil {
					currentUser = user.Username
					currentRole = user.Role
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"messages":    messages,
			"currentUser": currentUser,
			"currentRole": currentRole,
		})
	}
}

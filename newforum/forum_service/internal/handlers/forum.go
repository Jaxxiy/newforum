package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/jaxxiy/newforum/core/logger"
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
			return true
		},
	}
	clients   = make(map[int]map[*websocket.Conn]bool)
	clientsMu sync.RWMutex

	//mini-chat
	globalChatClients  = make(map[*websocket.Conn]bool)
	globalChatMu       sync.RWMutex
	globalChatUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	globalChatBroadcast = make(chan GlobalChatMessage)
	globalChatHistory   []GlobalChatMessage
	messageExpiration   = 1 * time.Minute

	authClient grpc.AuthClient
	log        = logger.GetLogger()
)

func init() {
	var err error
	authClient, err = grpc.NewClient("localhost:50051")
	if err != nil {
		log.Fatal("Failed to create auth client", logger.Error(err))
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
	// Start the cleanup routine
	go cleanupExpiredMessages(repo)

	r.HandleFunc("/ws/global", func(w http.ResponseWriter, r *http.Request) {
		serveGlobalChat(w, r, repo)
	})

	r.HandleFunc("/ws/{forum_id:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
		serveWebSocket(w, r)
	})
	go handleGlobalChatMessages()

	api := r.PathPrefix("/api").Subrouter()

	r.HandleFunc("/auth/login", LoginPage).Methods("GET")
	r.HandleFunc("/auth/register", RegisterPage).Methods("GET")

	api.HandleFunc("/forums", ListForums(repo)).Methods("GET")
	api.HandleFunc("/forums/new", NewForumForm()).Methods("GET")
	api.HandleFunc("/forums", CreateForum(repo)).Methods("POST")
	api.HandleFunc("/forums/{id:[0-9]+}", GetForum(repo)).Methods("GET")
	api.HandleFunc("/forums/{id:[0-9]+}", UpdateForum(repo)).Methods("PUT")
	api.HandleFunc("/forums/{id:[0-9]+}", DeleteForum(repo)).Methods("DELETE")

	api.HandleFunc("/forums/{id:[0-9]+}/messages", GetMessages(repo)).Methods("GET")
	api.HandleFunc("/forums/{id:[0-9]+}/messages", PostMessage(repo)).Methods("POST")
	api.HandleFunc("/forums/{forum_id:[0-9]+}/messages/{message_id:[0-9]+}", DeleteMessage(repo)).Methods("DELETE")
	api.HandleFunc("/forums/{id:[0-9]+}/messages/{message_id:[0-9]+}", UpdateMessage(repo)).Methods("PUT")

	api.HandleFunc("/global-chat", handleGlobalChatMessage(repo)).Methods("POST")

	api.HandleFunc("/forums/{id:[0-9]+}/messages-list", GetMessagesAPI(repo)).Methods("GET")

}

func LoginPage(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "login.html", nil)
}

func RegisterPage(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "register.html", nil)
}

func serveWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	forumID, err := strconv.Atoi(vars["forum_id"])
	if err != nil {
		http.Error(w, "Invalid forum ID", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("WebSocket upgrade error", logger.Error(err))
		return
	}
	defer func() {
		unregisterClient(forumID, conn)
		conn.Close()
	}()

	registerClient(forumID, conn)

	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Error("WebSocket error", logger.Error(err))
			}
			break
		}
	}
}

func broadcastToForum(forumID int, message WSMessage) {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	if conns, ok := clients[forumID]; ok {
		for conn := range conns {
			if err := conn.WriteJSON(message); err != nil {
				log.Error("WS send error",
					logger.Error(err),
					logger.Int("forumID", forumID))
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
		log.Info("Connection removed", logger.Int("forumID", forumID))
	}
}

func registerClient(forumID int, conn *websocket.Conn) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	if clients[forumID] == nil {
		clients[forumID] = make(map[*websocket.Conn]bool)
	}
	clients[forumID][conn] = true
	log.Info("New client connected",
		logger.Int("forumID", forumID),
		logger.Int("totalClients", len(clients[forumID])))
}

func unregisterClient(forumID int, conn *websocket.Conn) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	if clients[forumID] != nil {
		delete(clients[forumID], conn)
	}
}

func PostMessage(repo repository.ForumsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		forumID, err := strconv.Atoi(vars["id"])
		if err != nil {
			sendError(w, http.StatusBadRequest, "Invalid forum ID")
			return
		}

		if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			sendError(w, http.StatusBadRequest, "Content-Type must be application/json")
			return
		}

		var req struct {
			Author  string `json:"author"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid JSON format")
			return
		}

		if strings.TrimSpace(req.Author) == "" || strings.TrimSpace(req.Content) == "" {
			sendError(w, http.StatusBadRequest, "Author and content are required")
			return
		}

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

		if user.Username != req.Author && user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		msg := models.Message{
			ForumID:   forumID,
			Author:    req.Author,
			Content:   req.Content,
			CreatedAt: time.Now(),
		}

		fmt.Println(msg.CreatedAt)

		id, err := repo.CreateMessage(msg)
		if err != nil {
			log.Error("DB error", logger.Error(err))
			sendError(w, http.StatusInternalServerError, "Failed to save message")
			return
		}
		msg.ID = id

		go broadcastToForum(forumID, WSMessage{
			Type:    "message_created",
			Payload: msg,
		})

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(msg)
	}
}

func sendError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func sendWSMessage(forumID int, message WSMessage) {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	if conns, ok := clients[forumID]; ok {
		for conn := range conns {
			if err := conn.WriteJSON(message); err != nil {
				log.Error("WebSocket send error", logger.Error(err))
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
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Invalid forum ID", http.StatusBadRequest)
			return
		}

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

		forum, err := repo.GetByID(forumID)
		if err != nil {
			http.Error(w, "Forum not found", http.StatusNotFound)
			return
		}

		messages, err := repo.GetMessages(forumID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

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
		vars := mux.Vars(r)
		messageID, err := strconv.Atoi(vars["message_id"])
		fmt.Println(messageID)
		if err != nil {
			http.Error(w, "Invalid message ID", http.StatusBadRequest)
			return
		}

		var request struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

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

		msg, err := repo.GetMessageByID(messageID)
		if err != nil {
			http.Error(w, "Message not found", http.StatusNotFound)
			return
		}

		if user.Username != msg.Author && user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		updatedMessage, err := repo.PutMessage(messageID, request.Content)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

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

		msg, err := repo.GetMessageByID(messageID)
		if err != nil {
			http.Error(w, "Message not found", http.StatusNotFound)
			return
		}

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
		log.Error("Global chat WebSocket upgrade error", logger.Error(err))
		return
	}

	log.Info("New WebSocket connection established")

	defer func() {
		log.Info("WebSocket connection closed")
		globalChatMu.Lock()
		delete(globalChatClients, conn)
		globalChatMu.Unlock()
		conn.Close()
	}()

	globalChatMu.Lock()
	globalChatClients[conn] = true
	globalChatMu.Unlock()

	history, err := repo.GetGlobalChatHistory(100)
	if err != nil {
		log.Error("Error loading chat history", logger.Error(err))
	} else {
		for _, msg := range history {
			chatMsg := GlobalChatMessage{
				Author:    msg.Author,
				Content:   msg.Content,
				CreatedAt: msg.CreatedAt,
			}
			if err := conn.WriteJSON(chatMsg); err != nil {
				log.Error("Error sending history", logger.Error(err))
				break
			}
		}
	}

	for {
		var msg GlobalChatMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Error("Global chat error", logger.Error(err))
			}
			break
		}

		_, err := repo.CreateGlobalMessage(models.GlobalMessage{
			Author:    msg.Author,
			Content:   msg.Content,
			CreatedAt: time.Now(),
		})
		if err != nil {
			log.Error("Error saving message", logger.Error(err))
		}

		globalChatBroadcast <- msg
	}
}

func cleanupExpiredMessages(repo repository.ForumsRepository) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		globalChatMu.Lock()

		// Cleanup memory storage
		var activeMessages []GlobalChatMessage
		for _, msg := range globalChatHistory {
			if now.Sub(msg.CreatedAt) < messageExpiration {
				activeMessages = append(activeMessages, msg)
			}
		}
		globalChatHistory = activeMessages

		// Broadcast cleanup to all clients
		for client := range globalChatClients {
			if err := client.WriteJSON(WSMessage{
				Type: "cleanup",
				Payload: map[string]interface{}{
					"expiration": messageExpiration.Seconds(),
				},
			}); err != nil {
				log.Error("Cleanup broadcast error", logger.Error(err))
				client.Close()
				delete(globalChatClients, client)
			}
		}
		globalChatMu.Unlock()
	}
}

func handleGlobalChatMessages() {
	for {
		msg := <-globalChatBroadcast
		globalChatMu.Lock()

		if time.Since(msg.CreatedAt) < messageExpiration {
			globalChatHistory = append(globalChatHistory, msg)
			if len(globalChatHistory) > 100 {
				globalChatHistory = globalChatHistory[1:]
			}

			for client := range globalChatClients {
				if err := client.WriteJSON(msg); err != nil {
					log.Error("Error sending message", logger.Error(err))
					client.Close()
					delete(globalChatClients, client)
				}
			}
		}
		globalChatMu.Unlock()
	}
}

func handleGlobalChatMessage(repo repository.ForumsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, `{"error": "Content-Type must be application/json"}`, http.StatusBadRequest)
			return
		}

		var req GlobalChatMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "Invalid JSON format"}`, http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		if strings.TrimSpace(req.Author) == "" || strings.TrimSpace(req.Content) == "" {
			http.Error(w, `{"error": "Username and text are required"}`, http.StatusBadRequest)
			return
		}

		msgmodels := models.GlobalMessage{
			Author:    req.Author,
			Content:   req.Content,
			CreatedAt: time.Now(),
		}

		id, err := repo.CreateGlobalMessage(msgmodels)
		if err != nil {
			log.Error("DB error", logger.Error(err))
			http.Error(w, `{"error": "Failed to save message"}`, http.StatusInternalServerError)
			return
		}

		msgWebSocket := GlobalChatMessage{
			Author:    req.Author,
			Content:   req.Content,
			CreatedAt: time.Now(),
		}
		log.Info("Sending message to websocket", logger.String("author", req.Author), logger.String("text", req.Content))

		globalChatBroadcast <- msgWebSocket

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

		var currentUser, currentRole string
		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString != "" && authClient != nil {
				user, err := authClient.GetUserByToken(ctx, tokenString)
				if err != nil {
					log.Error("Error getting user by token", logger.Error(err))
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

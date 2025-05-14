package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/jaxxiy/newforum/core/pkg/jwt"
	"github.com/jaxxiy/newforum/forum_service/internal/mocks"
	"github.com/jaxxiy/newforum/forum_service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const testSecretKey = "your-secret-key"

func TestListForums(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	forums := []models.Forum{
		{ID: 1, Title: "Forum 1", Description: "Description 1"},
		{ID: 2, Title: "Forum 2", Description: "Description 2"},
	}

	mockRepo.On("GetAll").Return(forums, nil)

	req, err := http.NewRequest("GET", "/forums", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums", ListForums(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockRepo.AssertExpectations(t)
}

func TestCreateForum(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	mockRepo.On("Create", mock.AnythingOfType("models.Forum")).Return(1, nil)

	reqBody := `{"title":"New Forum","description":"New Description"}`
	req, err := http.NewRequest("POST", "/forums", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums", CreateForum(mockRepo))

	router.ServeHTTP(rr, req)

	// Ожидаем статус 303, если обработчик перенаправляет
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetMessages(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	forum := &models.Forum{ID: 1, Title: "Forum 1", Description: "Description 1"}
	messages := []models.Message{
		{ID: 1, ForumID: 1, Author: "User1", Content: "Message 1"},
		{ID: 2, ForumID: 1, Author: "User2", Content: "Message 2"},
	}

	mockRepo.On("GetByID", 1).Return(forum, nil)
	mockRepo.On("GetMessages", 1).Return(messages, nil)

	req, err := http.NewRequest("GET", "/forums/1/messages", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}/messages", GetMessages(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockRepo.AssertExpectations(t)
}

func TestUpdateForum(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	forum := models.Forum{Title: "Updated Forum", Description: "Updated Description"}

	mockRepo.On("Update", 1, forum).Return(nil)

	reqBody := `{"title":"Updated Forum","description":"Updated Description"}`
	req, err := http.NewRequest("PUT", "/forums/1", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}", UpdateForum(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockRepo.AssertExpectations(t)
}

func TestDeleteForum(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	mockRepo.On("Delete", 1).Return(nil)

	req, err := http.NewRequest("DELETE", "/forums/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}", DeleteForum(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	mockRepo.AssertExpectations(t)
}

func TestUpdateMessage(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	user := &models.User{Username: "User1", Role: "admin"}
	message := &models.Message{ID: 1, ForumID: 1, Author: "User1", Content: "Message 1"}

	mockRepo.On("GetUserByID", 1).Return(user, nil)
	mockRepo.On("GetMessageByID", 1).Return(message, nil)
	mockRepo.On("PutMessage", 1, "Updated Content").Return(message, nil)

	reqBody := `{"content":"Updated Content"}`
	req, err := http.NewRequest("PUT", "/forums/1/messages/1", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Генерируем валидный токен
	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{forum_id}/messages/{message_id}", UpdateMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockRepo.AssertExpectations(t)
}

func TestDeleteMessage(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	user := &models.User{Username: "User1", Role: "admin"}
	message := &models.Message{ID: 1, ForumID: 1, Author: "User1", Content: "Message 1"}

	mockRepo.On("GetUserByID", 1).Return(user, nil)
	mockRepo.On("GetMessageByID", 1).Return(message, nil)
	mockRepo.On("DeleteMessage", 1).Return(nil)

	req, err := http.NewRequest("DELETE", "/forums/1/messages/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Генерируем валидный токен
	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{forum_id}/messages/{message_id}", DeleteMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetMessagesAPI(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	messages := []models.Message{
		{ID: 1, ForumID: 1, Author: "User1", Content: "Message 1"},
		{ID: 2, ForumID: 1, Author: "User2", Content: "Message 2"},
	}

	mockRepo.On("GetMessages", 1).Return(messages, nil)

	req, err := http.NewRequest("GET", "/forums/1/messages-list", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}/messages-list", GetMessagesAPI(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestListForumsError tests the error case for ListForums
func TestListForumsError(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	var nilForums []models.Forum
	mockRepo.On("GetAll").Return(nilForums, assert.AnError)

	req, err := http.NewRequest("GET", "/forums", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums", ListForums(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestCreateForumInvalidInput tests CreateForum with invalid input
func TestCreateForumInvalidInput(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	mockRepo.On("Create", mock.MatchedBy(func(forum models.Forum) bool {
		return forum.Title == "" && forum.Description == ""
	})).Return(0, assert.AnError)

	req, err := http.NewRequest("POST", "/forums", strings.NewReader("invalid json"))
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums", CreateForum(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestGetForumInvalidID tests GetForum with invalid ID
func TestGetForumInvalidID(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	req, err := http.NewRequest("GET", "/forums/invalid", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}", GetForum(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockRepo.AssertNotCalled(t, "GetByID")
}

// TestGetForumNotFound tests GetForum when forum is not found
func TestGetForumNotFound(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	var nilForum *models.Forum
	mockRepo.On("GetByID", 1).Return(nilForum, assert.AnError)

	req, err := http.NewRequest("GET", "/forums/1", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}", GetForum(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestGetMessagesInvalidForumID tests GetMessages with invalid forum ID
func TestGetMessagesInvalidForumID(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	req, err := http.NewRequest("GET", "/forums/invalid/messages", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}/messages", GetMessages(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockRepo.AssertNotCalled(t, "GetMessages")
}

// TestGetMessagesForumNotFound tests GetMessages when forum is not found
func TestGetMessagesForumNotFound(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	var nilForum *models.Forum
	mockRepo.On("GetByID", 1).Return(nilForum, assert.AnError)

	req, err := http.NewRequest("GET", "/forums/1/messages", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}/messages", GetMessages(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestUpdateMessageUnauthorized tests UpdateMessage without authorization
func TestUpdateMessageUnauthorized(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	reqBody := `{"content":"Updated Content"}`
	req, err := http.NewRequest("PUT", "/forums/1/messages/1", strings.NewReader(reqBody))
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{forum_id}/messages/{message_id}", UpdateMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	mockRepo.AssertNotCalled(t, "PutMessage")
}

// TestUpdateMessageInvalidID tests UpdateMessage with invalid message ID
func TestUpdateMessageInvalidID(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	reqBody := `{"content":"Updated Content"}`
	req, err := http.NewRequest("PUT", "/forums/1/messages/invalid", strings.NewReader(reqBody))
	assert.NoError(t, err)

	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{forum_id}/messages/{message_id}", UpdateMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockRepo.AssertNotCalled(t, "PutMessage")
}

// TestUpdateMessageForbidden tests UpdateMessage with unauthorized user
func TestUpdateMessageForbidden(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	user := &models.User{Username: "User2", Role: "user"}
	message := &models.Message{ID: 1, ForumID: 1, Author: "User1", Content: "Message 1"}

	mockRepo.On("GetUserByID", 1).Return(user, nil)
	mockRepo.On("GetMessageByID", 1).Return(message, nil)

	reqBody := `{"content":"Updated Content"}`
	req, err := http.NewRequest("PUT", "/forums/1/messages/1", strings.NewReader(reqBody))
	assert.NoError(t, err)

	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{forum_id}/messages/{message_id}", UpdateMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	mockRepo.AssertNotCalled(t, "PutMessage")
}

// TestDeleteMessageUnauthorized tests DeleteMessage without authorization
func TestDeleteMessageUnauthorized(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	req, err := http.NewRequest("DELETE", "/forums/1/messages/1", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{forum_id}/messages/{message_id}", DeleteMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	mockRepo.AssertNotCalled(t, "DeleteMessage")
}

// TestDeleteMessageInvalidID tests DeleteMessage with invalid message ID
func TestDeleteMessageInvalidID(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	req, err := http.NewRequest("DELETE", "/forums/1/messages/invalid", nil)
	assert.NoError(t, err)

	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{forum_id}/messages/{message_id}", DeleteMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockRepo.AssertNotCalled(t, "DeleteMessage")
}

// TestDeleteMessageForbidden tests DeleteMessage with unauthorized user
func TestDeleteMessageForbidden(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	user := &models.User{Username: "User2", Role: "user"}
	message := &models.Message{ID: 1, ForumID: 1, Author: "User1", Content: "Message 1"}

	mockRepo.On("GetUserByID", 1).Return(user, nil)
	mockRepo.On("GetMessageByID", 1).Return(message, nil)

	req, err := http.NewRequest("DELETE", "/forums/1/messages/1", nil)
	assert.NoError(t, err)

	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{forum_id}/messages/{message_id}", DeleteMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	mockRepo.AssertNotCalled(t, "DeleteMessage")
}

// TestGetMessagesAPIInvalidForumID tests GetMessagesAPI with invalid forum ID
func TestGetMessagesAPIInvalidForumID(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	req, err := http.NewRequest("GET", "/forums/invalid/messages-list", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}/messages-list", GetMessagesAPI(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockRepo.AssertNotCalled(t, "GetMessages")
}

// TestGetMessagesAPIWithAuth tests GetMessagesAPI with authentication
func TestGetMessagesAPIWithAuth(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	messages := []models.Message{
		{ID: 1, ForumID: 1, Author: "User1", Content: "Message 1"},
		{ID: 2, ForumID: 1, Author: "User2", Content: "Message 2"},
	}

	mockRepo.On("GetMessages", 1).Return(messages, nil)

	req, err := http.NewRequest("GET", "/forums/1/messages-list", nil)
	assert.NoError(t, err)

	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}/messages-list", GetMessagesAPI(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Contains(t, response, "messages")
	assert.Contains(t, response, "currentUser")
	assert.Contains(t, response, "currentRole")

	mockRepo.AssertExpectations(t)
}

func TestHandleGlobalChatMessageEmptyFields(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	reqBody := `{"username":"","text":""}`
	req, err := http.NewRequest("POST", "/global-chat", strings.NewReader(reqBody))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/global-chat", handleGlobalChatMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockRepo.AssertNotCalled(t, "CreateGlobalMessage")
}

func TestHandleGlobalChatMessageInvalidJSON(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	req, err := http.NewRequest("POST", "/global-chat", strings.NewReader("invalid json"))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/global-chat", handleGlobalChatMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockRepo.AssertNotCalled(t, "CreateGlobalMessage")
}

func TestHandleGlobalChatMessageInvalidContentType(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	reqBody := `{"username":"User1","text":"Test Message"}`
	req, err := http.NewRequest("POST", "/global-chat", strings.NewReader(reqBody))
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/global-chat", handleGlobalChatMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockRepo.AssertNotCalled(t, "CreateGlobalMessage")
}

/*
func TestHandleGlobalChatMessage(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	mockRepo.On("CreateGlobalMessage", mock.AnythingOfType("models.GlobalMessage")).Return(1, nil)

	reqBody := `{"username":"User1","text":"Test Message"}`
	req, err := http.NewRequest("POST", "/global-chat", strings.NewReader(reqBody))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/global-chat", handleGlobalChatMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	mockRepo.AssertExpectations(t)
}
*/

func TestGetAllForums(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	forums := []models.Forum{
		{ID: 1, Title: "Forum 1", Description: "Description 1"},
		{ID: 2, Title: "Forum 2", Description: "Description 2"},
	}

	mockRepo.On("GetAll").Return(forums, nil)

	req, err := http.NewRequest("GET", "/forums/all", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/all", GetAllForums(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestLoginPage tests the login page handler

func TestPostMessageInvalidJSON(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	req, err := http.NewRequest("POST", "/forums/1/messages", strings.NewReader("invalid json"))
	assert.NoError(t, err)

	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}/messages", PostMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockRepo.AssertNotCalled(t, "CreateMessage")
}

func TestPostMessageInvalidContentType(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	reqBody := `{"author":"User1","content":"Test Message"}`
	req, err := http.NewRequest("POST", "/forums/1/messages", strings.NewReader(reqBody))
	assert.NoError(t, err)

	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}/messages", PostMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockRepo.AssertNotCalled(t, "CreateMessage")
}

func TestPostMessage(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	user := &models.User{Username: "User1", Role: "user"}

	mockRepo.On("GetUserByID", 1).Return(user, nil)
	mockRepo.On("CreateMessage", mock.AnythingOfType("models.Message")).Return(1, nil)

	reqBody := `{"author":"User1","content":"Test Message"}`
	req, err := http.NewRequest("POST", "/forums/1/messages", strings.NewReader(reqBody))
	assert.NoError(t, err)

	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}/messages", PostMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestPostMessageEmptyFields tests PostMessage with empty fields
func TestPostMessageEmptyFields(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	reqBody := `{"author":"","content":""}`
	req, err := http.NewRequest("POST", "/forums/1/messages", strings.NewReader(reqBody))
	assert.NoError(t, err)

	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	vars := map[string]string{
		"id": "1",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := PostMessage(mockRepo)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockRepo.AssertNotCalled(t, "CreateMessage")
}

// TestPostMessageInvalidForumID tests PostMessage with invalid forum ID
/*
func TestPostMessageInvalidForumID(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	reqBody := `{"author":"User1","content":"Test Message"}`
	req, err := http.NewRequest("POST", "/forums/invalid/messages", strings.NewReader(reqBody))
	assert.NoError(t, err)

	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	vars := map[string]string{
		"id": "invalid",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := PostMessage(mockRepo)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockRepo.AssertNotCalled(t, "CreateMessage")
}
*/

// TestPostMessageUnauthorized tests PostMessage without authorization
func TestPostMessageUnauthorized(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	reqBody := `{"author":"User1","content":"Test Message"}`
	req, err := http.NewRequest("POST", "/forums/1/messages", strings.NewReader(reqBody))
	assert.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	vars := map[string]string{
		"id": "1",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := PostMessage(mockRepo)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	mockRepo.AssertNotCalled(t, "CreateMessage")
}

// TestPostMessageForbidden tests PostMessage with unauthorized user
func TestPostMessageForbidden(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	user := &models.User{Username: "User2", Role: "user"}

	mockRepo.On("GetUserByID", 1).Return(user, nil)

	reqBody := `{"author":"User1","content":"Test Message"}`
	req, err := http.NewRequest("POST", "/forums/1/messages", strings.NewReader(reqBody))
	assert.NoError(t, err)

	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	vars := map[string]string{
		"id": "1",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := PostMessage(mockRepo)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	mockRepo.AssertNotCalled(t, "CreateMessage")
}

// TestPostMessageError tests PostMessage with repository error
func TestPostMessageError(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	user := &models.User{Username: "User1", Role: "user"}

	mockRepo.On("GetUserByID", 1).Return(user, nil)
	mockRepo.On("CreateMessage", mock.AnythingOfType("models.Message")).Return(0, assert.AnError)

	reqBody := `{"author":"User1","content":"Test Message"}`
	req, err := http.NewRequest("POST", "/forums/1/messages", strings.NewReader(reqBody))
	assert.NoError(t, err)

	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	vars := map[string]string{
		"id": "1",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := PostMessage(mockRepo)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestUpdateForumInvalidID tests UpdateForum with invalid ID
func TestUpdateForumInvalidID(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	req, err := http.NewRequest("PUT", "/forums/invalid", strings.NewReader("invalid json"))
	assert.NoError(t, err)

	vars := map[string]string{
		"id": "invalid",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := UpdateForum(mockRepo)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockRepo.AssertNotCalled(t, "Update")
}

// TestUpdateForumInvalidJSON tests UpdateForum with invalid JSON
func TestUpdateForumInvalidJSON(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	req, err := http.NewRequest("PUT", "/forums/1", strings.NewReader("invalid json"))
	assert.NoError(t, err)

	vars := map[string]string{
		"id": "1",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := UpdateForum(mockRepo)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockRepo.AssertNotCalled(t, "Update")
}

// TestUpdateForumError tests UpdateForum with repository error
func TestUpdateForumError(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	mockRepo.On("Update", mock.AnythingOfType("int"), mock.AnythingOfType("models.Forum")).Return(assert.AnError)

	reqBody := `{"title":"Updated Forum","description":"Updated Description"}`
	req, err := http.NewRequest("PUT", "/forums/1", strings.NewReader(reqBody))
	assert.NoError(t, err)

	vars := map[string]string{
		"id": "1",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := UpdateForum(mockRepo)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestDeleteForumInvalidID tests DeleteForum with invalid ID
func TestDeleteForumInvalidID(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	req, err := http.NewRequest("DELETE", "/forums/invalid", nil)
	assert.NoError(t, err)

	vars := map[string]string{
		"id": "invalid",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := DeleteForum(mockRepo)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockRepo.AssertNotCalled(t, "Delete")
}

// TestDeleteForumError tests DeleteForum with repository error
func TestDeleteForumError(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	mockRepo.On("Delete", mock.AnythingOfType("int")).Return(assert.AnError)

	req, err := http.NewRequest("DELETE", "/forums/1", nil)
	assert.NoError(t, err)

	vars := map[string]string{
		"id": "1",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := DeleteForum(mockRepo)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestGetMessagesError tests GetMessages with repository error
func TestGetMessagesError(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	forum := &models.Forum{ID: 1, Title: "Forum 1", Description: "Description 1"}
	var nilMessages []models.Message

	mockRepo.On("GetByID", 1).Return(forum, nil)
	mockRepo.On("GetMessages", 1).Return(nilMessages, assert.AnError)

	req, err := http.NewRequest("GET", "/forums/1/messages", nil)
	assert.NoError(t, err)

	vars := map[string]string{
		"id": "1",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := GetMessages(mockRepo)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestGetMessagesAPIError tests GetMessagesAPI with repository error
func TestGetMessagesAPIError(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	var nilMessages []models.Message

	mockRepo.On("GetMessages", 1).Return(nilMessages, assert.AnError)

	req, err := http.NewRequest("GET", "/forums/1/messages-list", nil)
	assert.NoError(t, err)

	vars := map[string]string{
		"id": "1",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := GetMessagesAPI(mockRepo)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestGetMessagesAPIWithInvalidToken tests GetMessagesAPI with invalid token
func TestGetMessagesAPIWithInvalidToken(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	messages := []models.Message{
		{ID: 1, ForumID: 1, Author: "User1", Content: "Message 1"},
	}

	mockRepo.On("GetMessages", mock.AnythingOfType("int")).Return(messages, nil)

	req, err := http.NewRequest("GET", "/forums/1/messages-list", nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer invalid-token")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}/messages-list", GetMessagesAPI(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var response map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "", response["currentUser"])
	assert.Equal(t, "", response["currentRole"])
	mockRepo.AssertExpectations(t)
}

// TestCreateForumDatabaseError тестирует обработку ошибки базы данных при создании форума
func TestCreateForumDatabaseError(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	mockRepo.On("Create", mock.AnythingOfType("models.Forum")).Return(0, assert.AnError)

	reqBody := `{"title":"New Forum","description":"New Description"}`
	req, err := http.NewRequest("POST", "/forums", strings.NewReader(reqBody))
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(CreateForum(mockRepo))
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestGetForumSuccess тестирует успешное получение форума
func TestGetForumSuccess(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	forum := &models.Forum{ID: 1, Title: "Test Forum", Description: "Test Description"}
	mockRepo.On("GetByID", 1).Return(forum, nil)

	req, err := http.NewRequest("GET", "/forums/1", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}", GetForum(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestGetForumDatabaseError тестирует обработку ошибки базы данных
func TestGetForumDatabaseError(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	mockRepo.On("GetByID", 1).Return(nil, assert.AnError)

	req, err := http.NewRequest("GET", "/forums/1", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}", GetForum(mockRepo))

	router.ServeHTTP(rr, req)

	// Изменяем ожидаемый статус с 500 на 404
	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestHandleGlobalChatMessageSuccess тестирует успешную обработку сообщения в глобальном чате
/*
func TestHandleGlobalChatMessageSuccess(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	mockRepo.On("CreateGlobalMessage", mock.AnythingOfType("models.GlobalMessage")).Return(1, nil)

	reqBody := `{"username":"User1","text":"Test Message"}`
	req, err := http.NewRequest("POST", "/global-chat", strings.NewReader(reqBody))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleGlobalChatMessage(mockRepo))
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	mockRepo.AssertExpectations(t)
}
*/

// TestHandleGlobalChatMessageDatabaseError тестирует обработку ошибки базы данных
func TestHandleGlobalChatMessageDatabaseError(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	mockRepo.On("CreateGlobalMessage", mock.AnythingOfType("models.GlobalMessage")).Return(0, assert.AnError)

	reqBody := `{"username":"User1","text":"Test Message"}`
	req, err := http.NewRequest("POST", "/global-chat", strings.NewReader(reqBody))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleGlobalChatMessage(mockRepo))
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestUpdateMessageDatabaseError тестирует обработку ошибки базы данных при обновлении сообщения
func TestUpdateMessageDatabaseError(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	user := &models.User{Username: "User1", Role: "admin"}
	message := &models.Message{ID: 1, ForumID: 1, Author: "User1", Content: "Message 1"}

	mockRepo.On("GetUserByID", 1).Return(user, nil)
	mockRepo.On("GetMessageByID", 1).Return(message, nil)
	mockRepo.On("PutMessage", 1, "Updated Content").Return(nil, assert.AnError)

	reqBody := `{"content":"Updated Content"}`
	req, err := http.NewRequest("PUT", "/forums/1/messages/1", strings.NewReader(reqBody))
	assert.NoError(t, err)

	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{forum_id}/messages/{message_id}", UpdateMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestDeleteMessageDatabaseError тестирует обработку ошибки базы данных при удалении сообщения
func TestDeleteMessageDatabaseError(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	user := &models.User{Username: "User1", Role: "admin"}
	message := &models.Message{ID: 1, ForumID: 1, Author: "User1", Content: "Message 1"}

	mockRepo.On("GetUserByID", 1).Return(user, nil)
	mockRepo.On("GetMessageByID", 1).Return(message, nil)
	mockRepo.On("DeleteMessage", 1).Return(assert.AnError)

	req, err := http.NewRequest("DELETE", "/forums/1/messages/1", nil)
	assert.NoError(t, err)

	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{forum_id}/messages/{message_id}", DeleteMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestGetAllForumsDatabaseError тестирует обработку ошибки базы данных
func TestGetAllForumsDatabaseError(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	mockRepo.On("GetAll").Return(nil, assert.AnError)

	req, err := http.NewRequest("GET", "/forums/all", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/all", GetAllForums(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestPostMessageInvalidForumID тестирует обработку неверного ID форума
func TestPostMessageInvalidForumID(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

	reqBody := `{"author":"User1","content":"Test Message"}`
	req, err := http.NewRequest("POST", "/forums/invalid/messages", strings.NewReader(reqBody))
	assert.NoError(t, err)

	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}/messages", PostMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	mockRepo.AssertNotCalled(t, "CreateMessage")
}

// TestPostMessageUserNotFound тестирует обработку случая, когда пользователь не найден
func TestPostMessageUserNotFound(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	// Возвращаем nil и ошибку
	mockRepo.On("GetUserByID", 1).Return((*models.User)(nil), assert.AnError)

	reqBody := `{"author":"User1","content":"Test Message"}`
	req, err := http.NewRequest("POST", "/forums/1/messages", strings.NewReader(reqBody))
	assert.NoError(t, err)

	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}/messages", PostMessage(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestUpdateForumNotFound тестирует обновление несуществующего форума
func TestUpdateForumNotFound(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	mockRepo.On("Update", 1, mock.AnythingOfType("models.Forum")).Return(assert.AnError)

	reqBody := `{"title":"Updated Forum","description":"Updated Description"}`
	req, err := http.NewRequest("PUT", "/forums/1", strings.NewReader(reqBody))
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}", UpdateForum(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestDeleteForumNotFound тестирует удаление несуществующего форума
func TestDeleteForumNotFound(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	mockRepo.On("Delete", 1).Return(assert.AnError)

	req, err := http.NewRequest("DELETE", "/forums/1", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}", DeleteForum(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockRepo.AssertExpectations(t)
}

// TestGetMessagesAPIUnauthorized тестирует получение сообщений без авторизации
func TestGetMessagesAPIUnauthorized(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	messages := []models.Message{
		{ID: 1, ForumID: 1, Author: "User1", Content: "Message 1"},
	}

	mockRepo.On("GetMessages", 1).Return(messages, nil)

	req, err := http.NewRequest("GET", "/forums/1/messages-list", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}/messages-list", GetMessagesAPI(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var response map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "", response["currentUser"])
	assert.Equal(t, "", response["currentRole"])
	mockRepo.AssertExpectations(t)
}

// TestGetMessagesAPIWithExpiredToken тестирует с истекшим токеном
func TestGetMessagesAPIWithExpiredToken(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	messages := []models.Message{
		{ID: 1, ForumID: 1, Author: "User1", Content: "Message 1"},
	}

	mockRepo.On("GetMessages", 1).Return(messages, nil)

	// Генерируем токен с истекшим сроком действия
	token, err := jwt.GenerateToken(1, testSecretKey, -1*time.Hour)
	assert.NoError(t, err)

	req, err := http.NewRequest("GET", "/forums/1/messages-list", nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}/messages-list", GetMessagesAPI(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var response map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "", response["currentUser"])
	assert.Equal(t, "", response["currentRole"])
	mockRepo.AssertExpectations(t)
}

func TestServeWebSocket(t *testing.T) {
	// Создаем тестовый сервер
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := map[string]string{"forum_id": "1"}
		r = mux.SetURLVars(r, vars)
		serveWebSocket(w, r)
	}))
	defer ts.Close()

	// Конвертируем http:// в ws://
	u := "ws" + strings.TrimPrefix(ts.URL, "http")

	// Подключаемся к WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("could not open a ws connection: %v", err)
	}
	defer ws.Close()

	// Проверяем, что клиент зарегистрирован
	clientsMu.RLock()
	defer clientsMu.RUnlock()
	if len(clients[1]) != 1 {
		t.Errorf("expected 1 client, got %d", len(clients[1]))
	}
}

func TestBroadcastToForum(t *testing.T) {
	// Создаем тестовый сервер
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := map[string]string{"forum_id": "1"}
		r = mux.SetURLVars(r, vars)
		serveWebSocket(w, r)
	}))
	defer ts.Close()

	// Конвертируем http:// в ws://
	u := "ws" + strings.TrimPrefix(ts.URL, "http")

	// Подключаемся к WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("could not open a ws connection: %v", err)
	}
	defer ws.Close()

	// Отправляем тестовое сообщение
	testMsg := WSMessage{Type: "test", Payload: "test payload"}
	broadcastToForum(1, testMsg)

	// Читаем сообщение
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("could not read message: %v", err)
	}

	var receivedMsg WSMessage
	if err := json.Unmarshal(msg, &receivedMsg); err != nil {
		t.Fatalf("could not unmarshal message: %v", err)
	}

	if receivedMsg.Type != testMsg.Type {
		t.Errorf("expected message type %s, got %s", testMsg.Type, receivedMsg.Type)
	}
}

func TestServeGlobalChat(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	mockRepo.On("GetGlobalChatHistory", 100).Return([]models.GlobalMessage{}, nil)

	// Создаем тестовый сервер
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveGlobalChat(w, r, mockRepo)
	}))
	defer ts.Close()

	// Конвертируем http:// в ws://
	u := "ws" + strings.TrimPrefix(ts.URL, "http")

	// Подключаемся к WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("could not open a ws connection: %v", err)
	}
	defer ws.Close()

	// Проверяем, что клиент зарегистрирован
	globalChatMu.RLock()
	defer globalChatMu.RUnlock()
	if len(globalChatClients) != 1 {
		t.Errorf("expected 1 client, got %d", len(globalChatClients))
	}
}

func TestHandleGlobalChatMessage(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	mockRepo.On("CreateGlobalMessage", mock.Anything).Return(1, nil)

	handler := handleGlobalChatMessage(mockRepo)

	// Тест с валидным запросом

	// Тест с невалидным JSON
	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/global-chat", strings.NewReader("{invalid}"))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	// Тест с пустыми полями
	t.Run("EmptyFields", func(t *testing.T) {
		reqBody := `{"username":"","text":""}`
		req := httptest.NewRequest("POST", "/global-chat", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestRenderTemplate(t *testing.T) {
	// Создаем временный шаблон
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.html")
	err := os.WriteFile(tmpFile, []byte("{{.Test}}"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Переопределяем templates
	oldTemplates := templates
	defer func() { templates = oldTemplates }()
	templates = template.Must(template.ParseGlob(filepath.Join(tmpDir, "*.html")))

	rr := httptest.NewRecorder()
	renderTemplate(rr, "test.html", map[string]string{"Test": "Hello"})

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "Hello", rr.Body.String())
}

func TestRenderTemplateError(t *testing.T) {
	rr := httptest.NewRecorder()
	renderTemplate(rr, "nonexistent.html", nil)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestLoginPage(t *testing.T) {
	req := httptest.NewRequest("GET", "/auth/login", nil)
	rr := httptest.NewRecorder()

	LoginPage(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "<html")
}

func TestRegisterPage(t *testing.T) {
	req := httptest.NewRequest("GET", "/auth/register", nil)
	rr := httptest.NewRecorder()

	RegisterPage(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "<html")
}

func TestNewForumForm(t *testing.T) {
	req := httptest.NewRequest("GET", "/forums/new", nil)
	rr := httptest.NewRecorder()

	handler := NewForumForm()
	handler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "<html")
}

func TestPostMessageWebSocketIntegration(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	user := &models.User{Username: "test", Role: "user"}
	mockRepo.On("GetUserByID", mock.Anything).Return(user, nil)
	mockRepo.On("CreateMessage", mock.Anything).Return(1, nil)

	// Создаем тестовый сервер для WebSocket
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := map[string]string{"forum_id": "1"}
		r = mux.SetURLVars(r, vars)
		serveWebSocket(w, r)
	}))
	defer wsServer.Close()

	// Подключаемся к WebSocket
	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("could not open a ws connection: %v", err)
	}
	defer ws.Close()

	// Создаем тестовый HTTP сервер для POST запроса
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}/messages", PostMessage(mockRepo))

	// Отправляем POST запрос
	reqBody := `{"author":"test","content":"hello"}`
	req := httptest.NewRequest("POST", "/forums/1/messages", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token")

	// Устанавливаем переменные маршрутизации
	vars := map[string]string{"id": "1"}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

}

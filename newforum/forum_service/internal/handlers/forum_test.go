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
func TestGetForumDatabaseError(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	mockRepo.On("GetByID", 1).Return(nil, assert.AnError)

	req, err := http.NewRequest("GET", "/forums/1", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}", GetForum(mockRepo))

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockRepo.AssertExpectations(t)
}
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
func TestPostMessageUserNotFound(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)

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
func TestGetMessagesAPIWithExpiredToken(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	messages := []models.Message{
		{ID: 1, ForumID: 1, Author: "User1", Content: "Message 1"},
	}

	mockRepo.On("GetMessages", 1).Return(messages, nil)

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

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := map[string]string{"forum_id": "1"}
		r = mux.SetURLVars(r, vars)
		serveWebSocket(w, r)
	}))
	defer ts.Close()

	u := "ws" + strings.TrimPrefix(ts.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("could not open a ws connection: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	clientsMu.RLock()
	clientCount := len(clients[1])
	clientsMu.RUnlock()

	if clientCount != 1 {
		t.Errorf("expected 1 client, got %d", clientCount)
	}

	err = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		t.Logf("error sending close message: %v", err)
	}
	ws.Close()

	time.Sleep(100 * time.Millisecond)

	clientsMu.RLock()
	clientCount = len(clients[1])
	clientsMu.RUnlock()

	if clientCount != 0 {
		t.Errorf("expected 0 clients after close, got %d", clientCount)
	}
}

func TestBroadcastToForum(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := map[string]string{"forum_id": "1"}
		r = mux.SetURLVars(r, vars)
		serveWebSocket(w, r)
	}))
	defer ts.Close()

	u := "ws" + strings.TrimPrefix(ts.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("could not open a ws connection: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	testMsg := WSMessage{Type: "test", Payload: "test payload"}
	broadcastToForum(1, testMsg)

	ws.SetReadDeadline(time.Now().Add(time.Second))
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

	err = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		t.Logf("error sending close message: %v", err)
	}
	ws.Close()
}

func TestServeGlobalChat(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	mockRepo.On("GetGlobalChatHistory", 100).Return([]models.GlobalMessage{}, nil)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveGlobalChat(w, r, mockRepo)
	}))
	defer ts.Close()

	u := "ws" + strings.TrimPrefix(ts.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("could not open a ws connection: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	globalChatMu.RLock()
	clientCount := len(globalChatClients)
	globalChatMu.RUnlock()

	if clientCount != 1 {
		t.Errorf("expected 1 client, got %d", clientCount)
	}

	testMsg := GlobalChatMessage{
		Author:    "test",
		Content:   "test message",
		CreatedAt: time.Now(),
	}
	err = ws.WriteJSON(testMsg)
	if err != nil {
		t.Fatalf("could not send message: %v", err)
	}

	err = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		t.Logf("error sending close message: %v", err)
	}
	ws.Close()

	mockRepo.AssertExpectations(t)
}

func TestPostMessageWebSocketIntegration(t *testing.T) {
	mockRepo := new(mocks.MockForumsRepo)
	user := &models.User{Username: "test", Role: "user"}
	mockRepo.On("GetUserByID", mock.Anything).Return(user, nil)
	mockRepo.On("CreateMessage", mock.Anything).Return(1, nil)

	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := map[string]string{"forum_id": "1"}
		r = mux.SetURLVars(r, vars)
		serveWebSocket(w, r)
	}))
	defer wsServer.Close()

	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("could not open a ws connection: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	router := mux.NewRouter()
	router.HandleFunc("/forums/{id}/messages", PostMessage(mockRepo))

	token, err := jwt.GenerateToken(1, testSecretKey, 24*time.Hour)
	if err != nil {
		t.Fatalf("could not generate token: %v", err)
	}

	reqBody := `{"author":"test","content":"hello"}`
	req := httptest.NewRequest("POST", "/forums/1/messages", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	vars := map[string]string{"id": "1"}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	ws.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("could not read message: %v", err)
	}

	var wsMsg WSMessage
	if err := json.Unmarshal(msg, &wsMsg); err != nil {
		t.Fatalf("could not unmarshal message: %v", err)
	}

	assert.Equal(t, "message_created", wsMsg.Type)

	err = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		t.Logf("error sending close message: %v", err)
	}
	ws.Close()

	mockRepo.AssertExpectations(t)
}

func TestRenderTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.html")
	err := os.WriteFile(tmpFile, []byte("{{.Test}}"), 0644)
	if err != nil {
		t.Fatal(err)
	}

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

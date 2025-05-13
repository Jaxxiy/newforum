package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
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

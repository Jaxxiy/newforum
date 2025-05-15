package service

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jaxxiy/newforum/forum_service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockForumRepo is a mock implementation of repository.ForumsRepository
type MockForumRepo struct {
	mock.Mock
}

func (m *MockForumRepo) GetAll() ([]models.Forum, error) {
	args := m.Called()
	return args.Get(0).([]models.Forum), args.Error(1)
}

func (m *MockForumRepo) GetByID(id int) (*models.Forum, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Forum), args.Error(1)
}

func (m *MockForumRepo) Create(forum models.Forum) (int, error) {
	args := m.Called(forum)
	return args.Int(0), args.Error(1)
}

func (m *MockForumRepo) Update(id int, forum models.Forum) error {
	args := m.Called(id, forum)
	return args.Error(0)
}

func (m *MockForumRepo) Delete(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockForumRepo) GetMessages(forumID int) ([]models.Message, error) {
	args := m.Called(forumID)
	return args.Get(0).([]models.Message), args.Error(1)
}

func (m *MockForumRepo) CreateMessage(message models.Message) (int, error) {
	args := m.Called(message)
	return args.Int(0), args.Error(1)
}

func (m *MockForumRepo) GetMessageByID(id int) (*models.Message, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Message), args.Error(1)
}

func (m *MockForumRepo) PutMessage(id int, content string) (*models.Message, error) {
	args := m.Called(id, content)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Message), args.Error(1)
}

func (m *MockForumRepo) DeleteMessage(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockForumRepo) GetUserByID(id int) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockForumRepo) CreateGlobalMessage(message models.GlobalMessage) (int, error) {
	args := m.Called(message)
	return args.Int(0), args.Error(1)
}

func (m *MockForumRepo) GetGlobalChatHistory(limit int) ([]models.GlobalMessage, error) {
	args := m.Called(limit)
	return args.Get(0).([]models.GlobalMessage), args.Error(1)
}

func TestNewForumService(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)
	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
}

func TestGetAllForums(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	expectedForums := []models.Forum{
		{ID: 1, Title: "Forum 1", Description: "Description 1"},
		{ID: 2, Title: "Forum 2", Description: "Description 2"},
	}

	mockRepo.On("GetAll").Return(expectedForums, nil)

	forums, err := service.GetAllForums()
	assert.NoError(t, err)
	assert.Equal(t, expectedForums, forums)
	mockRepo.AssertExpectations(t)
}

func TestGetForumByID(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	expectedForum := &models.Forum{ID: 1, Title: "Forum 1", Description: "Description 1"}
	mockRepo.On("GetByID", 1).Return(expectedForum, nil)

	forum, err := service.GetForumByID(1)
	assert.NoError(t, err)
	assert.Equal(t, expectedForum, forum)
	mockRepo.AssertExpectations(t)
}

func TestCreateForum(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	forum := models.Forum{Title: "New Forum", Description: "New Description"}
	expectedID := 1

	mockRepo.On("Create", forum).Return(expectedID, nil)

	id, err := service.CreateForum(forum)
	assert.NoError(t, err)
	assert.Equal(t, expectedID, id)
	mockRepo.AssertExpectations(t)
}

func TestUpdateForum(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	forum := models.Forum{Title: "Updated Forum", Description: "Updated Description"}
	mockRepo.On("Update", 1, forum).Return(nil)

	err := service.UpdateForum(1, forum)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestDeleteForum(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	mockRepo.On("Delete", 1).Return(nil)

	err := service.DeleteForum(1)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGetMessages(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	expectedMessages := []models.Message{
		{ID: 1, ForumID: 1, Author: "User1", Content: "Message 1"},
		{ID: 2, ForumID: 1, Author: "User2", Content: "Message 2"},
	}

	mockRepo.On("GetMessages", 1).Return(expectedMessages, nil)

	messages, err := service.GetMessages(1)
	assert.NoError(t, err)
	assert.Equal(t, expectedMessages, messages)
	mockRepo.AssertExpectations(t)
}

func TestCreateMessage(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	message := models.Message{
		ForumID: 1,
		Author:  "User1",
		Content: "New Message",
	}
	expectedID := 1

	mockRepo.On("CreateMessage", message).Return(expectedID, nil)

	id, err := service.CreateMessage(message)
	assert.NoError(t, err)
	assert.Equal(t, expectedID, id)
	mockRepo.AssertExpectations(t)
}

func TestGetMessageByID(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	expectedMessage := &models.Message{
		ID:      1,
		ForumID: 1,
		Author:  "User1",
		Content: "Message 1",
	}

	mockRepo.On("GetMessageByID", 1).Return(expectedMessage, nil)

	message, err := service.GetMessageByID(1)
	assert.NoError(t, err)
	assert.Equal(t, expectedMessage, message)
	mockRepo.AssertExpectations(t)
}

func TestUpdateMessage(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	expectedMessage := &models.Message{
		ID:      1,
		ForumID: 1,
		Author:  "User1",
		Content: "Updated Message",
	}

	mockRepo.On("PutMessage", 1, "Updated Message").Return(expectedMessage, nil)

	message, err := service.UpdateMessage(1, "Updated Message")
	assert.NoError(t, err)
	assert.Equal(t, expectedMessage, message)
	mockRepo.AssertExpectations(t)
}

func TestDeleteMessage(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	mockRepo.On("DeleteMessage", 1).Return(nil)

	err := service.DeleteMessage(1)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGetUserByID(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	expectedUser := &models.User{
		ID:       1,
		Username: "User1",
		Role:     "user",
	}

	mockRepo.On("GetUserByID", 1).Return(expectedUser, nil)

	user, err := service.GetUserByID(1)
	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
	mockRepo.AssertExpectations(t)
}

func TestCreateGlobalMessage(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	message := models.GlobalMessage{
		Author:    "User1",
		Content:   "Global Message",
		CreatedAt: time.Now(),
	}
	expectedID := 1

	mockRepo.On("CreateGlobalMessage", message).Return(expectedID, nil)

	id, err := service.CreateGlobalMessage(message)
	assert.NoError(t, err)
	assert.Equal(t, expectedID, id)
	mockRepo.AssertExpectations(t)
}

func TestGetGlobalChatHistory(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	expectedMessages := []models.GlobalMessage{
		{ID: 1, Author: "User1", Content: "Message 1"},
		{ID: 2, Author: "User2", Content: "Message 2"},
	}

	mockRepo.On("GetGlobalChatHistory", 100).Return(expectedMessages, nil)

	messages, err := service.GetGlobalChatHistory(100)
	assert.NoError(t, err)
	assert.Equal(t, expectedMessages, messages)
	mockRepo.AssertExpectations(t)
}

func TestErrorCases(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	t.Run("GetAllForums Error", func(t *testing.T) {
		mockRepo.On("GetAll").Return([]models.Forum{}, assert.AnError)
		_, err := service.GetAllForums()
		assert.Error(t, err)
	})

	t.Run("GetForumByID Error", func(t *testing.T) {
		mockRepo.On("GetByID", 1).Return((*models.Forum)(nil), assert.AnError)
		_, err := service.GetForumByID(1)
		assert.Error(t, err)
	})

	t.Run("CreateForum Error", func(t *testing.T) {
		forum := models.Forum{Title: "Error Forum"}
		mockRepo.On("Create", forum).Return(0, assert.AnError)
		_, err := service.CreateForum(forum)
		assert.Error(t, err)
	})

	t.Run("UpdateForum Error", func(t *testing.T) {
		forum := models.Forum{Title: "Error Forum"}
		mockRepo.On("Update", 1, forum).Return(assert.AnError)
		err := service.UpdateForum(1, forum)
		assert.Error(t, err)
	})

	t.Run("DeleteForum Error", func(t *testing.T) {
		mockRepo.On("Delete", 1).Return(assert.AnError)
		err := service.DeleteForum(1)
		assert.Error(t, err)
	})

	t.Run("GetMessages Error", func(t *testing.T) {
		mockRepo.On("GetMessages", 1).Return([]models.Message{}, assert.AnError)
		_, err := service.GetMessages(1)
		assert.Error(t, err)
	})

	t.Run("CreateMessage Error", func(t *testing.T) {
		message := models.Message{ForumID: 1, Author: "User1"}
		mockRepo.On("CreateMessage", message).Return(0, assert.AnError)
		_, err := service.CreateMessage(message)
		assert.Error(t, err)
	})

	t.Run("GetMessageByID Error", func(t *testing.T) {
		mockRepo.On("GetMessageByID", 1).Return((*models.Message)(nil), assert.AnError)
		_, err := service.GetMessageByID(1)
		assert.Error(t, err)
	})

	t.Run("UpdateMessage Error", func(t *testing.T) {
		mockRepo.On("PutMessage", 1, "Error").Return((*models.Message)(nil), assert.AnError)
		_, err := service.UpdateMessage(1, "Error")
		assert.Error(t, err)
	})

	t.Run("DeleteMessage Error", func(t *testing.T) {
		mockRepo.On("DeleteMessage", 1).Return(assert.AnError)
		err := service.DeleteMessage(1)
		assert.Error(t, err)
	})

	t.Run("GetUserByID Error", func(t *testing.T) {
		mockRepo.On("GetUserByID", 1).Return((*models.User)(nil), assert.AnError)
		_, err := service.GetUserByID(1)
		assert.Error(t, err)
	})

	t.Run("CreateGlobalMessage Error", func(t *testing.T) {
		message := models.GlobalMessage{Author: "User1"}
		mockRepo.On("CreateGlobalMessage", message).Return(0, assert.AnError)
		_, err := service.CreateGlobalMessage(message)
		assert.Error(t, err)
	})

	t.Run("GetGlobalChatHistory Error", func(t *testing.T) {
		mockRepo.On("GetGlobalChatHistory", 100).Return([]models.GlobalMessage{}, assert.AnError)
		_, err := service.GetGlobalChatHistory(100)
		assert.Error(t, err)
	})
}

func TestForumValidation(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	t.Run("Empty Title", func(t *testing.T) {
		forum := models.Forum{
			Title:       "",
			Description: "Description",
		}
		_, err := service.CreateForum(forum)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "title")
	})

	t.Run("Empty Description", func(t *testing.T) {
		forum := models.Forum{
			Title:       "Title",
			Description: "",
		}
		_, err := service.CreateForum(forum)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "description")
	})

	t.Run("Title Too Long", func(t *testing.T) {
		forum := models.Forum{
			Title:       strings.Repeat("a", 256),
			Description: "Description",
		}
		_, err := service.CreateForum(forum)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "title too long")
	})
}

func TestMessageValidation(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	t.Run("Empty Content", func(t *testing.T) {
		message := models.Message{
			ForumID: 1,
			Author:  "User1",
			Content: "",
		}
		_, err := service.CreateMessage(message)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content")
	})

	t.Run("Empty Author", func(t *testing.T) {
		message := models.Message{
			ForumID: 1,
			Author:  "",
			Content: "Content",
		}
		_, err := service.CreateMessage(message)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "author")
	})

	t.Run("Invalid ForumID", func(t *testing.T) {
		message := models.Message{
			ForumID: 0,
			Author:  "User1",
			Content: "Content",
		}
		_, err := service.CreateMessage(message)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forum ID")
	})
}

func TestGlobalMessageValidation(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	t.Run("Empty Content", func(t *testing.T) {
		message := models.GlobalMessage{
			Author:  "User1",
			Content: "",
		}
		_, err := service.CreateGlobalMessage(message)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content")
	})

	t.Run("Empty Author", func(t *testing.T) {
		message := models.GlobalMessage{
			Author:  "",
			Content: "Content",
		}
		_, err := service.CreateGlobalMessage(message)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "author")
	})

	t.Run("Content Too Long", func(t *testing.T) {
		message := models.GlobalMessage{
			Author:  "User1",
			Content: strings.Repeat("a", 5001),
		}
		_, err := service.CreateGlobalMessage(message)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content too long")
	})
}

func TestGetMessagesWithPagination(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	// Create test messages
	messages := make([]models.Message, 20)
	for i := 0; i < 20; i++ {
		messages[i] = models.Message{
			ID:      i + 1,
			ForumID: 1,
			Author:  fmt.Sprintf("User%d", i+1),
			Content: fmt.Sprintf("Message %d", i+1),
		}
	}

	mockRepo.On("GetMessages", 1).Return(messages, nil)

	// Test getting all messages
	result, err := service.GetMessages(1)
	assert.NoError(t, err)
	assert.Len(t, result, 20)

	// Test error case
	mockRepo.On("GetMessages", 999).Return([]models.Message{}, assert.AnError)
	_, err = service.GetMessages(999)
	assert.Error(t, err)
}

func TestUserOperations(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	t.Run("Get Existing User", func(t *testing.T) {
		expectedUser := &models.User{
			ID:       1,
			Username: "testuser",
			Role:     "user",
		}
		mockRepo.On("GetUserByID", 1).Return(expectedUser, nil)

		user, err := service.GetUserByID(1)
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, user)
	})

	t.Run("Get Non-existent User", func(t *testing.T) {
		mockRepo.On("GetUserByID", 999).Return((*models.User)(nil), assert.AnError)

		user, err := service.GetUserByID(999)
		assert.Error(t, err)
		assert.Nil(t, user)
	})

	t.Run("Invalid User ID", func(t *testing.T) {
		user, err := service.GetUserByID(0)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "invalid user ID")
	})
}

func TestForumOperationsWithTransaction(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	t.Run("Create Forum with Messages", func(t *testing.T) {
		forum := models.Forum{
			Title:       "Test Forum",
			Description: "Test Description",
		}
		mockRepo.On("Create", forum).Return(1, nil)

		message := models.Message{
			ForumID: 1,
			Author:  "User1",
			Content: "First message",
		}
		mockRepo.On("CreateMessage", message).Return(1, nil)

		forumID, err := service.CreateForum(forum)
		assert.NoError(t, err)
		assert.Equal(t, 1, forumID)

		messageID, err := service.CreateMessage(message)
		assert.NoError(t, err)
		assert.Equal(t, 1, messageID)
	})

	t.Run("Delete Forum with Messages", func(t *testing.T) {
		mockRepo.On("Delete", 1).Return(nil)
		err := service.DeleteForum(1)
		assert.NoError(t, err)
	})
}

func TestGlobalChatOperations(t *testing.T) {
	mockRepo := new(MockForumRepo)
	service := NewForumService(mockRepo)

	t.Run("Get Chat History with Limit", func(t *testing.T) {
		messages := []models.GlobalMessage{
			{ID: 1, Author: "User1", Content: "Message 1"},
			{ID: 2, Author: "User2", Content: "Message 2"},
		}
		mockRepo.On("GetGlobalChatHistory", 100).Return(messages, nil)

		result, err := service.GetGlobalChatHistory(100)
		assert.NoError(t, err)
		assert.Equal(t, messages, result)
	})

	t.Run("Invalid History Limit", func(t *testing.T) {
		_, err := service.GetGlobalChatHistory(0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid limit")
	})

	t.Run("Get History Error", func(t *testing.T) {
		mockRepo.On("GetGlobalChatHistory", 50).Return([]models.GlobalMessage{}, assert.AnError)
		_, err := service.GetGlobalChatHistory(50)
		assert.Error(t, err)
	})
}

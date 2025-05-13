package mocks

import (
	"github.com/jaxxiy/newforum/forum_service/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockForumsRepo реализует интерфейс repository.ForumsRepository
type MockForumsRepo struct {
	mock.Mock
}

func (m *MockForumsRepo) GetAll() ([]models.Forum, error) {
	args := m.Called()
	return args.Get(0).([]models.Forum), args.Error(1)
}

func (m *MockForumsRepo) GetByID(id int) (*models.Forum, error) {
	args := m.Called(id)
	return args.Get(0).(*models.Forum), args.Error(1)
}

func (m *MockForumsRepo) Create(forum models.Forum) (int, error) {
	args := m.Called(forum)
	return args.Int(0), args.Error(1)
}

func (m *MockForumsRepo) Update(id int, forum models.Forum) error {
	args := m.Called(id, forum)
	return args.Error(0)
}

func (m *MockForumsRepo) Delete(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockForumsRepo) GetMessages(forumID int) ([]models.Message, error) {
	args := m.Called(forumID)
	return args.Get(0).([]models.Message), args.Error(1)
}

func (m *MockForumsRepo) CreateMessage(msg models.Message) (int, error) {
	args := m.Called(msg)
	return args.Int(0), args.Error(1)
}

func (m *MockForumsRepo) GetMessageByID(id int) (*models.Message, error) {
	args := m.Called(id)
	return args.Get(0).(*models.Message), args.Error(1)
}

func (m *MockForumsRepo) PutMessage(id int, content string) (*models.Message, error) {
	args := m.Called(id, content)
	return args.Get(0).(*models.Message), args.Error(1)
}

func (m *MockForumsRepo) DeleteMessage(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockForumsRepo) CreateGlobalMessage(msg models.GlobalMessage) (int, error) {
	args := m.Called(msg)
	return args.Int(0), args.Error(1)
}

func (m *MockForumsRepo) GetGlobalChatHistory(limit int) ([]models.GlobalMessage, error) {
	args := m.Called(limit)
	return args.Get(0).([]models.GlobalMessage), args.Error(1)
}

func (m *MockForumsRepo) GetUserByID(id int) (*models.User, error) {
	args := m.Called(id)
	return args.Get(0).(*models.User), args.Error(1)
}

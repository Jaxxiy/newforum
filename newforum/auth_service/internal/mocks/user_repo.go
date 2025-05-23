package mocks

import (
	"github.com/jaxxiy/newforum/auth_service/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) Create(user models.User) (int, error) {
	args := m.Called(user)
	return args.Int(0), args.Error(1)
}

func (m *MockUserRepo) GetByUsername(username string) (*models.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepo) GetByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepo) GetUserByID(userID int) (*models.User, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepo) UpdatePassword(userID int, hashedPassword string) error {
	args := m.Called(userID, hashedPassword)
	return args.Error(0)
}

func (m *MockUserRepo) SetupSuccessfulCreate(userID int) {
	m.On("Create", mock.AnythingOfType("models.User")).
		Return(userID, nil)
}

func (m *MockUserRepo) SetupCreateError(err error) {
	m.On("Create", mock.AnythingOfType("models.User")).
		Return(0, err)
}

func (m *MockUserRepo) SetupGetByUsername(username string, user *models.User, err error) {
	m.On("GetByUsername", username).
		Return(user, err)
}

func (m *MockUserRepo) SetupGetByEmail(email string, user *models.User, err error) {
	m.On("GetByEmail", email).
		Return(user, err)
}

func (m *MockUserRepo) SetupGetUserByID(userID int, user *models.User, err error) {
	m.On("GetUserByID", userID).
		Return(user, err)
}

func (m *MockUserRepo) SetupUpdatePassword(userID int, err error) {
	m.On("UpdatePassword", userID, mock.AnythingOfType("string")).
		Return(err)
}

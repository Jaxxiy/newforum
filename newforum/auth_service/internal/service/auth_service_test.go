package service

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jaxxiy/newforum/auth_service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// MockUserRepo implements repository.UserRepository
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

func TestNewAuthService(t *testing.T) {
	mockRepo := &MockUserRepo{}
	service := NewAuthService(mockRepo)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.userRepo)
	assert.NotNil(t, service.jwtKey)
}

func TestAuthService_Register(t *testing.T) {
	mockRepo := &MockUserRepo{}
	service := NewAuthService(mockRepo)

	tests := []struct {
		name          string
		request       models.RegisterRequest
		setupMocks    func()
		expectedError string
	}{
		{
			name: "successful registration",
			request: models.RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMocks: func() {
				mockRepo.On("GetByUsername", "testuser").Return(nil, errors.New("not found"))
				mockRepo.On("GetByEmail", "test@example.com").Return(nil, errors.New("not found"))
				mockRepo.On("Create", mock.AnythingOfType("models.User")).Return(1, nil)
			},
		},
		{
			name: "username exists",
			request: models.RegisterRequest{
				Username: "existinguser",
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMocks: func() {
				mockRepo.On("GetByUsername", "existinguser").Return(&models.User{}, nil)
			},
			expectedError: "username already exists",
		},
		{
			name: "email exists",
			request: models.RegisterRequest{
				Username: "testuser",
				Email:    "existing@example.com",
				Password: "password123",
			},
			setupMocks: func() {
				mockRepo.On("GetByUsername", "testuser").Return(nil, errors.New("not found"))
				mockRepo.On("GetByEmail", "existing@example.com").Return(&models.User{}, nil)
			},
			expectedError: "email already exists",
		},
		{
			name: "create user error",
			request: models.RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMocks: func() {
				mockRepo.On("GetByUsername", "testuser").Return(nil, errors.New("not found"))
				mockRepo.On("GetByEmail", "test@example.com").Return(nil, errors.New("not found"))
				mockRepo.On("Create", mock.AnythingOfType("models.User")).Return(0, errors.New("db error"))
			},
			expectedError: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.ExpectedCalls = nil
			mockRepo.Calls = nil
			tt.setupMocks()

			response, err := service.Register(tt.request)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotEmpty(t, response.Token)
				assert.Equal(t, tt.request.Username, response.User.Username)
				assert.Equal(t, tt.request.Email, response.User.Email)
				assert.Equal(t, "user", response.User.Role)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	mockRepo := &MockUserRepo{}
	service := NewAuthService(mockRepo)

	// Create a test user with properly hashed password
	password := "password123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	testUser := &models.User{
		ID:        1,
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  string(hashedPassword),
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tests := []struct {
		name          string
		request       models.LoginRequest
		setupMocks    func()
		expectedError string
	}{
		{
			name: "successful login",
			request: models.LoginRequest{
				Username: "testuser",
				Password: password,
			},
			setupMocks: func() {
				mockRepo.On("GetByUsername", "testuser").Return(testUser, nil)
			},
		},
		{
			name: "user not found",
			request: models.LoginRequest{
				Username: "nonexistent",
				Password: "password123",
			},
			setupMocks: func() {
				mockRepo.On("GetByUsername", "nonexistent").Return(nil, sql.ErrNoRows)
			},
			expectedError: "invalid username or password",
		},
		{
			name: "wrong password",
			request: models.LoginRequest{
				Username: "testuser",
				Password: "wrongpassword",
			},
			setupMocks: func() {
				mockRepo.On("GetByUsername", "testuser").Return(testUser, nil)
			},
			expectedError: "invalid username or password",
		},
		{
			name: "database error",
			request: models.LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			setupMocks: func() {
				mockRepo.On("GetByUsername", "testuser").Return(nil, errors.New("db error"))
			},
			expectedError: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.ExpectedCalls = nil
			mockRepo.Calls = nil
			tt.setupMocks()

			response, err := service.Login(tt.request)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotEmpty(t, response.Token)
				assert.Equal(t, testUser.Username, response.User.Username)
				assert.Equal(t, testUser.Email, response.User.Email)
				assert.Equal(t, testUser.Role, response.User.Role)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	mockRepo := &MockUserRepo{}
	service := NewAuthService(mockRepo)

	testUser := &models.User{
		ID:        1,
		Username:  "testuser",
		Email:     "test@example.com",
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Generate a valid token
	token, err := service.generateToken(*testUser)
	assert.NoError(t, err)

	tests := []struct {
		name          string
		token         string
		setupMocks    func()
		expectedError string
	}{
		{
			name:  "valid token",
			token: token,
			setupMocks: func() {
				mockRepo.On("GetUserByID", 1).Return(testUser, nil)
			},
		},
		{
			name:          "invalid token format",
			token:         "invalid.token.format",
			setupMocks:    func() {},
			expectedError: "invalid character",
		},
		{
			name:  "user not found",
			token: token,
			setupMocks: func() {
				mockRepo.On("GetUserByID", 1).Return(nil, sql.ErrNoRows)
			},
			expectedError: "sql: no rows in result set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.ExpectedCalls = nil
			mockRepo.Calls = nil
			tt.setupMocks()

			user, err := service.ValidateToken(tt.token)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, testUser.Username, user.Username)
				assert.Equal(t, testUser.Email, user.Email)
				assert.Equal(t, testUser.Role, user.Role)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_GetUserByID(t *testing.T) {
	mockRepo := &MockUserRepo{}
	service := NewAuthService(mockRepo)

	testUser := &models.User{
		ID:        1,
		Username:  "testuser",
		Email:     "test@example.com",
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tests := []struct {
		name          string
		userID        int
		setupMocks    func()
		expectedError string
	}{
		{
			name:   "user found",
			userID: 1,
			setupMocks: func() {
				mockRepo.On("GetUserByID", 1).Return(testUser, nil)
			},
		},
		{
			name:   "user not found",
			userID: 999,
			setupMocks: func() {
				mockRepo.On("GetUserByID", 999).Return(nil, sql.ErrNoRows)
			},
			expectedError: "sql: no rows in result set",
		},
		{
			name:   "database error",
			userID: 1,
			setupMocks: func() {
				mockRepo.On("GetUserByID", 1).Return(nil, errors.New("db error"))
			},
			expectedError: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.ExpectedCalls = nil
			mockRepo.Calls = nil
			tt.setupMocks()

			user, err := service.GetUserByID(tt.userID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, testUser.Username, user.Username)
				assert.Equal(t, testUser.Email, user.Email)
				assert.Equal(t, testUser.Role, user.Role)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

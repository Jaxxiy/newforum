package mocks

import (
	"context"
	"errors"
	"time"

	"github.com/jaxxiy/newforum/forum_service/pkg/models"
)

// MockAuthService represents a mock implementation of the auth service
type MockAuthService struct {
	users map[int]*models.User
}

// NewMockAuthService creates a new instance of MockAuthService
func NewMockAuthService() *MockAuthService {
	return &MockAuthService{
		users: make(map[int]*models.User),
	}
}

// Register mocks user registration
func (m *MockAuthService) Register(req models.RegisterRequest) (*models.AuthResponse, error) {
	// Check if username already exists
	for _, user := range m.users {
		if user.Username == req.Username {
			return nil, errors.New("username already exists")
		}
		if user.Email == req.Email {
			return nil, errors.New("email already exists")
		}
	}

	// Create new user
	user := &models.User{
		ID:        len(m.users) + 1,
		Username:  req.Username,
		Email:     req.Email,
		Password:  "hashed_" + req.Password, // Simulate password hashing
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	m.users[user.ID] = user

	return &models.AuthResponse{
		Token: "mock_token_" + user.Username,
		User:  *user,
	}, nil
}

// Login mocks user login
func (m *MockAuthService) Login(req models.LoginRequest) (*models.AuthResponse, error) {
	for _, user := range m.users {
		if user.Username == req.Username {
			// Simulate password check
			if "hashed_"+req.Password == user.Password {
				return &models.AuthResponse{
					Token: "mock_token_" + user.Username,
					User:  *user,
				}, nil
			}
			return nil, errors.New("invalid username or password")
		}
	}
	return nil, errors.New("invalid username or password")
}

// ValidateToken mocks token validation
func (m *MockAuthService) ValidateToken(token string) (*models.User, error) {
	// Simple mock implementation that extracts username from token
	if len(token) > 11 && token[:11] == "mock_token_" {
		username := token[11:]
		for _, user := range m.users {
			if user.Username == username {
				return user, nil
			}
		}
	}
	return nil, errors.New("invalid token")
}

// GetUserByToken mocks getting user by token for gRPC interface
func (m *MockAuthService) GetUserByToken(ctx context.Context, token string) (*models.User, error) {
	return m.ValidateToken(token)
}

// AddMockUser adds a pre-configured user to the mock service
func (m *MockAuthService) AddMockUser(user *models.User) {
	m.users[user.ID] = user
}

package mocks

import (
	"context"
	"errors"
	"time"

	"github.com/jaxxiy/newforum/forum_service/pkg/models"
)

type MockAuthService struct {
	users map[int]*models.User
}

func NewMockAuthService() *MockAuthService {
	return &MockAuthService{
		users: make(map[int]*models.User),
	}
}

func (m *MockAuthService) Register(req models.RegisterRequest) (*models.AuthResponse, error) {
	for _, user := range m.users {
		if user.Username == req.Username {
			return nil, errors.New("username already exists")
		}
		if user.Email == req.Email {
			return nil, errors.New("email already exists")
		}
	}

	user := &models.User{
		ID:        len(m.users) + 1,
		Username:  req.Username,
		Email:     req.Email,
		Password:  "hashed_" + req.Password,
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

func (m *MockAuthService) Login(req models.LoginRequest) (*models.AuthResponse, error) {
	for _, user := range m.users {
		if user.Username == req.Username {
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

func (m *MockAuthService) ValidateToken(token string) (*models.User, error) {
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

func (m *MockAuthService) GetUserByToken(ctx context.Context, token string) (*models.User, error) {
	return m.ValidateToken(token)
}

func (m *MockAuthService) AddMockUser(user *models.User) {
	m.users[user.ID] = user
}

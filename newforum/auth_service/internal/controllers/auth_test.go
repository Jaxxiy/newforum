package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jaxxiy/newforum/auth_service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
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

func TestAuthController_Pages(t *testing.T) {
	tmpl := template.New("test")
	tmpl, err := tmpl.Parse(`{{define "register.html"}}Register{{end}} {{define "login.html"}}Login{{end}}`)
	assert.NoError(t, err)

	mockRepo := new(MockUserRepo)
	controller := &AuthController{
		userRepo:  mockRepo,
		jwtSecret: "test-secret",
		templates: tmpl,
	}

	t.Run("RegisterPage success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/register", nil)
		rr := httptest.NewRecorder()

		controller.RegisterPage(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Header().Get("Content-Type"), "text/html")
		assert.Equal(t, "Register", rr.Body.String())
	})

	t.Run("RegisterPage wrong method", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/register", nil)
		rr := httptest.NewRecorder()

		controller.RegisterPage(rr, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
		assert.Contains(t, rr.Body.String(), "Method not allowed")
	})

	t.Run("RegisterPage template error", func(t *testing.T) {
		brokenTmpl := template.New("broken")
		controllerWithBrokenTemplate := &AuthController{
			userRepo:  mockRepo,
			jwtSecret: "test-secret",
			templates: brokenTmpl,
		}

		req := httptest.NewRequest("GET", "/register", nil)
		rr := httptest.NewRecorder()

		controllerWithBrokenTemplate.RegisterPage(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "Template error")
	})

	t.Run("LoginPage success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/login", nil)
		rr := httptest.NewRecorder()

		controller.LoginPage(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Header().Get("Content-Type"), "text/html")
		assert.Equal(t, "Login", rr.Body.String())
	})

	t.Run("LoginPage wrong method", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/login", nil)
		rr := httptest.NewRecorder()

		controller.LoginPage(rr, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
		assert.Contains(t, rr.Body.String(), "Method not allowed")
	})

	t.Run("LoginPage template error", func(t *testing.T) {
		brokenTmpl := template.New("broken")
		controllerWithBrokenTemplate := &AuthController{
			userRepo:  mockRepo,
			jwtSecret: "test-secret",
			templates: brokenTmpl,
		}

		req := httptest.NewRequest("GET", "/login", nil)
		rr := httptest.NewRecorder()

		controllerWithBrokenTemplate.LoginPage(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "Template error")
	})
}

func TestAuthController_Register(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockReturn     int
		mockError      error
		expectedStatus int
		expectedBody   map[string]string
		method         string
	}{
		{
			name: "successful registration",
			requestBody: models.RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			mockReturn:     1,
			expectedStatus: http.StatusCreated,
			expectedBody:   map[string]string{"status": "user created"},
			method:         http.MethodPost,
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid",
			expectedStatus: http.StatusBadRequest,
			method:         http.MethodPost,
		},
		{
			name: "database error",
			requestBody: models.RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			mockError:      errors.New("db error"),
			expectedStatus: http.StatusInternalServerError,
			method:         http.MethodPost,
		},
		{
			name: "wrong method",
			requestBody: models.RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedStatus: http.StatusMethodNotAllowed,
			method:         http.MethodGet,
		},
		{
			name: "empty request body",
			requestBody: models.RegisterRequest{
				Username: "",
				Email:    "",
				Password: "",
			},
			expectedStatus: http.StatusBadRequest,
			method:         http.MethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepo)
			controller := &AuthController{
				userRepo:  mockRepo,
				jwtSecret: "test-secret",
				templates: template.New("test"),
			}

			if tt.mockReturn != 0 || tt.mockError != nil {
				mockRepo.On("Create", mock.AnythingOfType("models.User")).
					Return(tt.mockReturn, tt.mockError).Maybe()
			}

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(tt.method, "/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			controller.Register(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedBody != nil {
				var response map[string]string
				err := json.NewDecoder(rr.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody, response)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAuthController_Login(t *testing.T) {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	tests := []struct {
		name           string
		requestBody    interface{}
		mockUser       *models.User
		mockError      error
		expectedStatus int
		expectedKeys   []string
		method         string
	}{
		{
			name: "successful login",
			requestBody: models.LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			mockUser: &models.User{
				ID:        1,
				Username:  "testuser",
				Email:     "test@example.com",
				Password:  string(hashedPassword),
				Role:      "user",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectedStatus: http.StatusOK,
			expectedKeys:   []string{"token", "user_id", "username", "email"},
			method:         http.MethodPost,
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid",
			expectedStatus: http.StatusBadRequest,
			method:         http.MethodPost,
		},
		{
			name: "user not found",
			requestBody: models.LoginRequest{
				Username: "nonexistent",
				Password: "password123",
			},
			mockError:      errors.New("user not found"),
			expectedStatus: http.StatusUnauthorized,
			method:         http.MethodPost,
		},
		{
			name: "wrong password",
			requestBody: models.LoginRequest{
				Username: "testuser",
				Password: "wrongpass",
			},
			mockUser: &models.User{
				ID:        1,
				Username:  "testuser",
				Email:     "test@example.com",
				Password:  string(hashedPassword),
				Role:      "user",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectedStatus: http.StatusUnauthorized,
			method:         http.MethodPost,
		},
		{
			name: "wrong method",
			requestBody: models.LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			expectedStatus: http.StatusMethodNotAllowed,
			method:         http.MethodGet,
		},
		{
			name: "empty credentials",
			requestBody: models.LoginRequest{
				Username: "",
				Password: "",
			},
			expectedStatus: http.StatusBadRequest,
			method:         http.MethodPost,
		},
		{
			name:           "malformed json",
			requestBody:    `{"username": "testuser", "password": "password123"`, // incomplete JSON
			expectedStatus: http.StatusBadRequest,
			method:         http.MethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepo)
			controller := &AuthController{
				userRepo:  mockRepo,
				jwtSecret: "test-secret",
				templates: template.New("test"),
			}

			if tt.mockUser != nil || tt.mockError != nil {
				mockRepo.On("GetByUsername", mock.AnythingOfType("string")).
					Return(tt.mockUser, tt.mockError)
			}

			var body []byte
			if s, ok := tt.requestBody.(string); ok && tt.name == "malformed json" {
				body = []byte(s)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(tt.method, "/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			controller.Login(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedKeys != nil {
				var response map[string]interface{}
				err := json.NewDecoder(rr.Body).Decode(&response)
				assert.NoError(t, err)
				for _, key := range tt.expectedKeys {
					assert.Contains(t, response, key)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestNewAuthController(t *testing.T) {
	mockRepo := new(MockUserRepo)

	tmpl := template.New("test")
	tmpl, err := tmpl.Parse(`{{define "register.html"}}Register{{end}} {{define "login.html"}}Login{{end}}`)
	assert.NoError(t, err)

	controller := &AuthController{
		userRepo:  mockRepo,
		jwtSecret: "test-secret",
		templates: tmpl,
	}

	assert.NotNil(t, controller)
	assert.Equal(t, mockRepo, controller.userRepo)
	assert.Equal(t, "test-secret", controller.jwtSecret)
	assert.NotNil(t, controller.templates)
}

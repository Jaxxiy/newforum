package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/jaxxiy/newforum/auth_service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(req models.RegisterRequest) (*models.AuthResponse, error) {
	args := m.Called(req)
	return args.Get(0).(*models.AuthResponse), args.Error(1)
}

func (m *MockAuthService) Login(req models.LoginRequest) (*models.AuthResponse, error) {
	args := m.Called(req)
	return args.Get(0).(*models.AuthResponse), args.Error(1)
}

func (m *MockAuthService) ValidateToken(token string) (*models.User, error) {
	args := m.Called(token)
	return args.Get(0).(*models.User), args.Error(1)
}

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockReturn     *models.AuthResponse
		mockError      error
		expectedStatus int
		expectBody     bool
		expectedToken  string
	}{
		{
			name: "successful registration",
			requestBody: models.RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			mockReturn:     &models.AuthResponse{Token: "test-token"},
			expectedStatus: http.StatusCreated,
			expectBody:     true,
			expectedToken:  "test-token",
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid",
			expectedStatus: http.StatusBadRequest,
			expectBody:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAuthService)
			if tt.mockReturn != nil || tt.mockError != nil {
				mockService.On("Register", mock.AnythingOfType("models.RegisterRequest")).
					Return(tt.mockReturn, tt.mockError)
			}

			handler := NewAuthHandler(mockService)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.Register(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectBody {
				if tt.mockError != nil {

					var resp map[string]string
					err := json.NewDecoder(rr.Body).Decode(&resp)
					assert.NoError(t, err)
					assert.Equal(t, tt.mockError.Error(), resp["error"])
				} else {

					var resp models.AuthResponse
					err := json.NewDecoder(rr.Body).Decode(&resp)
					assert.NoError(t, err)
					assert.Equal(t, tt.expectedToken, resp.Token)
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}
func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockReturn     *models.AuthResponse
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful login",
			requestBody: models.LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			mockReturn:     &models.AuthResponse{Token: "test-token"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid json",
			requestBody:    "invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid credentials",
			requestBody: models.LoginRequest{
				Username: "testuser",
				Password: "wrongpass",
			},
			mockError:      errors.New("invalid credentials"),
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAuthService)
			if tt.mockReturn != nil || tt.mockError != nil {
				mockService.On("Login", mock.AnythingOfType("models.LoginRequest")).
					Return(tt.mockReturn, tt.mockError)
			}

			handler := NewAuthHandler(mockService)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.Login(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_ValidateToken(t *testing.T) {
	tests := []struct {
		name           string
		setupRequest   func(*http.Request)
		mockReturn     *models.User
		mockError      error
		expectedStatus int
	}{
		{
			name: "valid token",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer valid-token")
			},
			mockReturn: &models.User{
				Username: "testuser",
				Email:    "test@example.com",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing header",
			setupRequest:   func(req *http.Request) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid header format",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "InvalidFormat")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid token",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer invalid-token")
			},
			mockError:      errors.New("invalid token"),
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAuthService)
			if tt.mockReturn != nil || tt.mockError != nil {
				mockService.On("ValidateToken", mock.AnythingOfType("string")).
					Return(tt.mockReturn, tt.mockError)
			}

			handler := NewAuthHandler(mockService)

			req := httptest.NewRequest("GET", "/validate", nil)
			tt.setupRequest(req)
			rr := httptest.NewRecorder()

			handler.ValidateToken(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_RegisterPage(t *testing.T) {
	mockService := new(MockAuthService)
	handler := NewAuthHandler(mockService)

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/register", nil)
		rr := httptest.NewRecorder()

		handler.RegisterPage(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Header().Get("Content-Type"), "text/html")
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/register", nil)
		rr := httptest.NewRecorder()

		handler.RegisterPage(rr, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
	})
}

func TestAuthHandler_LoginPage(t *testing.T) {
	mockService := new(MockAuthService)
	handler := NewAuthHandler(mockService)

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/login", nil)
		rr := httptest.NewRecorder()

		handler.LoginPage(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Header().Get("Content-Type"), "text/html")
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/login", nil)
		rr := httptest.NewRecorder()

		handler.LoginPage(rr, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
	})
}

func TestRegisterAuthRoutes(t *testing.T) {
	router := mux.NewRouter()
	mockService := new(MockAuthService)
	handler := NewAuthHandler(mockService)

	RegisterAuthRoutes(router, handler)

	tests := []struct {
		method string
		path   string
	}{
		{"POST", "/auth/register"},
		{"POST", "/auth/login"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			match := &mux.RouteMatch{}
			assert.True(t, router.Match(req, match), "route not registered")
		})
	}
}

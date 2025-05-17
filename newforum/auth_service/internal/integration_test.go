package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"database/sql"

	"github.com/gorilla/mux"
	"github.com/jaxxiy/newforum/auth_service/internal/handlers"
	"github.com/jaxxiy/newforum/auth_service/internal/models"
	"github.com/jaxxiy/newforum/auth_service/internal/repository"
	"github.com/jaxxiy/newforum/auth_service/internal/service"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func setupTestServer(t *testing.T) (*mux.Router, func()) {
	// Use test database URL
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:Stas2005101010!@localhost:5432/forum_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Ensure the database is reachable
	if err := db.Ping(); err != nil {
		t.Fatalf("Could not ping test database: %v", err)
	}

	userRepo := repository.NewUserRepo(db)
	authService := service.NewAuthService(userRepo)
	authHandler := handlers.NewAuthHandler(authService)

	r := mux.NewRouter()
	auth := r.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/register", authHandler.Register).Methods("POST", "OPTIONS")
	auth.HandleFunc("/login", authHandler.Login).Methods("POST", "OPTIONS")

	cleanup := func() {
		// Clean up test data
		db.Exec("DELETE FROM users WHERE email = $1", "test@example.com")
		db.Close()
	}

	return r, cleanup
}

func TestAuthenticationFlow(t *testing.T) {
	router, cleanup := setupTestServer(t)
	defer cleanup()

	// Test user data
	testUser := models.RegisterRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "testPassword123!",
	}

	// Step 1: Test Registration
	t.Run("Registration", func(t *testing.T) {
		userJSON, err := json.Marshal(testUser)
		assert.NoError(t, err)
		fmt.Printf("Registration request body: %s\n", string(userJSON))

		req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(userJSON))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Read and log the response body
		body, err := ioutil.ReadAll(rr.Body)
		assert.NoError(t, err)
		fmt.Printf("Registration response: Status=%d, Body=%s\n", rr.Code, string(body))

		assert.Equal(t, http.StatusCreated, rr.Code)

		var response models.AuthResponse
		err = json.NewDecoder(bytes.NewBuffer(body)).Decode(&response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response.Token)
		assert.Equal(t, testUser.Username, response.User.Username)
		assert.Equal(t, testUser.Email, response.User.Email)
	})

	// Step 2: Test Login
	t.Run("Login", func(t *testing.T) {
		loginData := models.LoginRequest{
			Username: testUser.Username,
			Password: testUser.Password,
		}

		loginJSON, err := json.Marshal(loginData)
		assert.NoError(t, err)
		fmt.Printf("Login request body: %s\n", string(loginJSON))

		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginJSON))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Read and log the response body
		body, err := ioutil.ReadAll(rr.Body)
		assert.NoError(t, err)
		fmt.Printf("Login response: Status=%d, Body=%s\n", rr.Code, string(body))

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.AuthResponse
		err = json.NewDecoder(bytes.NewBuffer(body)).Decode(&response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response.Token)
		assert.Equal(t, testUser.Username, response.User.Username)
	})

	// Step 3: Test Login with Wrong Password
	t.Run("Login with Wrong Password", func(t *testing.T) {
		loginData := models.LoginRequest{
			Username: testUser.Username,
			Password: "wrongpassword",
		}

		loginJSON, err := json.Marshal(loginData)
		assert.NoError(t, err)
		fmt.Printf("Wrong password login request body: %s\n", string(loginJSON))

		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginJSON))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Read and log the response body
		body, err := ioutil.ReadAll(rr.Body)
		assert.NoError(t, err)
		fmt.Printf("Wrong password login response: Status=%d, Body=%s\n", rr.Code, string(body))

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

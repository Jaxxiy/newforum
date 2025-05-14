package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	secret := "test-secret"
	middleware := AuthMiddleware(secret)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id")
		if userID != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("user_id not found in context"))
		}
	})

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": float64(1), // JWT converts numbers to float64
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	validToken, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": float64(1),
		"exp":     time.Now().Add(-time.Hour).Unix(), // expired 1 hour ago
	})
	expiredTokenString, err := expiredToken.SignedString([]byte(secret))
	assert.NoError(t, err)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "valid token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "no auth header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized\n",
		},
		{
			name:           "invalid token format",
			authHeader:     "Bearer invalid.token.format",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid token\n",
		},
		{
			name:           "expired token",
			authHeader:     "Bearer " + expiredTokenString,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid token\n",
		},
		{
			name:           "malformed bearer token",
			authHeader:     "Bearertoken",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid token\n",
		},
		{
			name:           "wrong signing method",
			authHeader:     "Bearer " + "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJleHAiOjE3MTA4NzY5NTB9.ZGFrZQ",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid token\n",
		},
		{
			name:           "token signed with wrong key",
			authHeader:     "Bearer " + generateTokenWithDifferentSecret(t),
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid token\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			handler := middleware(nextHandler)
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
		})
	}
}

func generateTokenWithDifferentSecret(t *testing.T) string {
	wrongSecret := "wrong-secret"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": float64(1),
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(wrongSecret))
	assert.NoError(t, err)
	return tokenString
}

func TestAuthMiddleware_Integration(t *testing.T) {
	secret := "test-secret"
	middleware := AuthMiddleware(secret)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id")
		if userID == nil {
			t.Error("user_id not found in context")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var id int
		switch v := userID.(type) {
		case int:
			id = v
		case float64:
			id = int(v)
		default:
			t.Errorf("unexpected user_id type: got %T, want int or float64", userID)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if id != 1 {
			t.Errorf("unexpected user_id value: got %v, want 1", id)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": float64(1),
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	validToken, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	rr := httptest.NewRecorder()

	middlewareChain := middleware(handler)
	middlewareChain.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthMiddleware_ContextPropagation(t *testing.T) {
	secret := "test-secret"
	middleware := AuthMiddleware(secret)

	parentHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "parent_key", "parent_value")
		r = r.WithContext(ctx)
		middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parentValue := r.Context().Value("parent_key")
			userID := r.Context().Value("user_id")
			if parentValue != "parent_value" || userID == nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(w, r)
	})

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": float64(1),
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	validToken, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	rr := httptest.NewRecorder()

	parentHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

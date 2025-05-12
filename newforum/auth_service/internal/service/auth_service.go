package service

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jaxxiy/newforum/auth_service/internal/models"
	"github.com/jaxxiy/newforum/auth_service/internal/repository"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

type AuthServiceInterface interface {
	Register(req models.RegisterRequest) (*models.AuthResponse, error)
	Login(req models.LoginRequest) (*models.AuthResponse, error)
	ValidateToken(token string) (*models.User, error)
}

type AuthService struct {
	userRepo repository.UserRepository
	jwtKey   []byte
}

func NewAuthService(userRepo repository.UserRepository) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		jwtKey:   []byte("your-secret-key"),
	}
}

func (s *AuthService) Register(req models.RegisterRequest) (*models.AuthResponse, error) {
	// Check if username already exists
	if _, err := s.userRepo.GetByUsername(req.Username); err == nil {
		return nil, errors.New("username already exists")
	}

	// Check if email already exists
	if _, err := s.userRepo.GetByEmail(req.Email); err == nil {
		return nil, errors.New("email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user
	user := models.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(hashedPassword),
		Role:      "user", // всегда user
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	userID, err := s.userRepo.Create(user)
	if err != nil {
		return nil, err
	}
	user.ID = userID

	// Generate JWT token
	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

func (s *AuthService) Login(req models.LoginRequest) (*models.AuthResponse, error) {
	// Get user by username
	user, err := s.userRepo.GetByUsername(req.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("invalid username or password")
		}
		return nil, err
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid username or password")
	}

	// Generate JWT token
	token, err := s.generateToken(*user)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*models.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return s.jwtKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID := int(claims["user_id"].(float64))
		return s.userRepo.GetUserByID(userID)
	}

	return nil, errors.New("invalid token")
}

func (s *AuthService) generateToken(user models.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	return token.SignedString(s.jwtKey)
}

func (s *AuthService) GetUserByID(userID int) (*models.User, error) {
	return s.userRepo.GetUserByID(userID)
}

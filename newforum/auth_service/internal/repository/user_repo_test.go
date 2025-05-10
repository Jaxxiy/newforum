package repository

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jaxxiy/newforum/auth_service/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestUserRepo_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewUserRepo(db)

	tests := []struct {
		name    string
		user    models.User
		mock    func()
		want    int
		wantErr bool
	}{
		{
			name: "Success",
			user: models.User{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password",
				Role:     "user",
			},
			mock: func() {
				rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
				mock.ExpectQuery("INSERT INTO users").
					WithArgs("testuser", "test@example.com", "password", "user", sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
			want: 1,
		},
		{
			name: "Empty Fields",
			user: models.User{
				Username: "",
				Email:    "",
				Password: "",
				Role:     "",
			},
			mock: func() {
				mock.ExpectQuery("INSERT INTO users").
					WithArgs("", "", "", "", sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnError(errors.New("empty fields"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			got, err := repo.Create(tt.user)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Create() got = %v, want %v", got, tt.want)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepo_GetByUsername(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewUserRepo(db)

	testTime := time.Now()

	tests := []struct {
		name     string
		username string
		mock     func()
		want     *models.User
		wantErr  bool
	}{
		{
			name:     "Success",
			username: "testuser",
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "username", "email", "password", "role", "created_at", "updated_at"}).
					AddRow(1, "testuser", "test@example.com", "password", "user", testTime, testTime)
				mock.ExpectQuery("SELECT id, username, email, password, role, created_at, updated_at FROM users WHERE username = \\$1").
					WithArgs("testuser").
					WillReturnRows(rows)
			},
			want: &models.User{
				ID:        1,
				Username:  "testuser",
				Email:     "test@example.com",
				Password:  "password",
				Role:      "user",
				CreatedAt: testTime,
				UpdatedAt: testTime,
			},
		},
		{
			name:     "Not Found",
			username: "nonexistent",
			mock: func() {
				mock.ExpectQuery("SELECT id, username, email, password, role, created_at, updated_at FROM users WHERE username = \\$1").
					WithArgs("nonexistent").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			got, err := repo.GetByUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepo_GetByEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewUserRepo(db)

	testTime := time.Now()

	tests := []struct {
		name    string
		email   string
		mock    func()
		want    *models.User
		wantErr bool
	}{
		{
			name:  "Success",
			email: "test@example.com",
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "username", "email", "password", "role", "created_at", "updated_at"}).
					AddRow(1, "testuser", "test@example.com", "password", "user", testTime, testTime)
				mock.ExpectQuery("SELECT id, username, email, password, role, created_at, updated_at FROM users WHERE email = \\$1").
					WithArgs("test@example.com").
					WillReturnRows(rows)
			},
			want: &models.User{
				ID:        1,
				Username:  "testuser",
				Email:     "test@example.com",
				Password:  "password",
				Role:      "user",
				CreatedAt: testTime,
				UpdatedAt: testTime,
			},
		},
		{
			name:  "Not Found",
			email: "nonexistent@example.com",
			mock: func() {
				mock.ExpectQuery("SELECT id, username, email, password, role, created_at, updated_at FROM users WHERE email = \\$1").
					WithArgs("nonexistent@example.com").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			got, err := repo.GetByEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepo_GetUserByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewUserRepo(db)

	testTime := time.Now()

	tests := []struct {
		name    string
		userID  int
		mock    func()
		want    *models.User
		wantErr bool
	}{
		{
			name:   "Success",
			userID: 1,
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "username", "email", "role", "created_at", "updated_at"}).
					AddRow(1, "testuser", "test@example.com", "user", testTime, testTime)
				mock.ExpectQuery("SELECT id, username, email, role, created_at, updated_at FROM users WHERE id = \\$1").
					WithArgs(1).
					WillReturnRows(rows)
			},
			want: &models.User{
				ID:        1,
				Username:  "testuser",
				Email:     "test@example.com",
				Role:      "user",
				CreatedAt: testTime,
				UpdatedAt: testTime,
			},
		},
		{
			name:   "Not Found",
			userID: 999,
			mock: func() {
				mock.ExpectQuery("SELECT id, username, email, role, created_at, updated_at FROM users WHERE id = \\$1").
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			got, err := repo.GetUserByID(tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepo_UpdatePassword(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewUserRepo(db)

	tests := []struct {
		name           string
		userID         int
		hashedPassword string
		mock           func()
		wantErr        bool
		expectedErr    error
	}{
		{
			name:           "Success",
			userID:         1,
			hashedPassword: "newhashedpassword",
			mock: func() {
				mock.ExpectExec(`UPDATE users`).
					WithArgs("newhashedpassword", sqlmock.AnyArg(), 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr:     false,
			expectedErr: nil,
		},
		{
			name:           "User Not Found",
			userID:         999,
			hashedPassword: "newhashedpassword",
			mock: func() {
				mock.ExpectExec(`UPDATE users`).
					WithArgs("newhashedpassword", sqlmock.AnyArg(), 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr:     true,
			expectedErr: errors.New("user not found"),
		},
		{
			name:           "Database Error",
			userID:         1,
			hashedPassword: "newhashedpassword",
			mock: func() {
				mock.ExpectExec(`UPDATE users`).
					WithArgs("newhashedpassword", sqlmock.AnyArg(), 1).
					WillReturnError(errors.New("database error"))
			},
			wantErr:     true,
			expectedErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			err := repo.UpdatePassword(tt.userID, tt.hashedPassword)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if err.Error() != tt.expectedErr.Error() {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

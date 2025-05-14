package repository

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jaxxiy/newforum/forum_service/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestForumsRepo_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewForumsRepo(db)

	tests := []struct {
		name    string
		forum   models.Forum
		mock    func()
		want    int
		wantErr bool
	}{
		{
			name: "Success",
			forum: models.Forum{
				Title:       "Test Forum",
				Description: "Test Description",
			},
			mock: func() {
				rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
				mock.ExpectQuery(`INSERT INTO forums`).
					WithArgs("Test Forum", "Test Description").
					WillReturnRows(rows)
			},
			want: 1,
		},
		{
			name: "Database Error",
			forum: models.Forum{
				Title:       "Test Forum",
				Description: "Test Description",
			},
			mock: func() {
				mock.ExpectQuery(`INSERT INTO forums`).
					WithArgs("Test Forum", "Test Description").
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			got, err := repo.Create(tt.forum)
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

func TestForumsRepo_GetAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewForumsRepo(db)

	testTime := time.Now()

	tests := []struct {
		name    string
		mock    func()
		want    []models.Forum
		wantErr bool
	}{
		{
			name: "Success",
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "name", "description", "created_at"}).
					AddRow(1, "Forum 1", "Desc 1", testTime).
					AddRow(2, "Forum 2", "Desc 2", testTime)
				mock.ExpectQuery(`SELECT id, name, description, created_at FROM forums`).
					WillReturnRows(rows)
			},
			want: []models.Forum{
				{ID: 1, Title: "Forum 1", Description: "Desc 1", CreatedAt: testTime},
				{ID: 2, Title: "Forum 2", Description: "Desc 2", CreatedAt: testTime},
			},
		},
		{
			name: "Empty Result",
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "name", "description", "created_at"})
				mock.ExpectQuery(`SELECT id, name, description, created_at FROM forums`).
					WillReturnRows(rows)
			},
			want:    []models.Forum{},
			wantErr: false,
		},
		{
			name: "Database Error",
			mock: func() {
				mock.ExpectQuery(`SELECT id, name, description, created_at FROM forums`).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			got, err := repo.GetAll()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestForumsRepo_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewForumsRepo(db)

	tests := []struct {
		name    string
		id      int
		mock    func()
		want    *models.Forum
		wantErr bool
	}{
		{
			name: "Success",
			id:   1,
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "name", "description"}).
					AddRow(1, "Test Forum", "Test Description")
				mock.ExpectQuery(`SELECT id, name, description FROM forums WHERE id = \$1`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			want: &models.Forum{
				ID:          1,
				Title:       "Test Forum",
				Description: "Test Description",
			},
		},
		{
			name: "Not Found",
			id:   999,
			mock: func() {
				mock.ExpectQuery(`SELECT id, name, description FROM forums WHERE id = \$1`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name: "Database Error",
			id:   1,
			mock: func() {
				mock.ExpectQuery(`SELECT id, name, description FROM forums WHERE id = \$1`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			got, err := repo.GetByID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestForumsRepo_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewForumsRepo(db)

	tests := []struct {
		name    string
		id      int
		forum   models.Forum
		mock    func()
		wantErr bool
	}{
		{
			name: "Success",
			id:   1,
			forum: models.Forum{
				Title:       "Updated Forum",
				Description: "Updated Description",
			},
			mock: func() {
				mock.ExpectExec(`UPDATE forums`).
					WithArgs("Updated Forum", "Updated Description", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "Not Found",
			id:   999,
			forum: models.Forum{
				Title:       "Updated Forum",
				Description: "Updated Description",
			},
			mock: func() {
				mock.ExpectExec(`UPDATE forums`).
					WithArgs("Updated Forum", "Updated Description", 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
		},
		{
			name: "Database Error",
			id:   1,
			forum: models.Forum{
				Title:       "Updated Forum",
				Description: "Updated Description",
			},
			mock: func() {
				mock.ExpectExec(`UPDATE forums`).
					WithArgs("Updated Forum", "Updated Description", 1).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			err := repo.Update(tt.id, tt.forum)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestForumsRepo_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewForumsRepo(db)

	tests := []struct {
		name    string
		id      int
		mock    func()
		wantErr bool
	}{
		{
			name: "Success",
			id:   1,
			mock: func() {
				mock.ExpectExec(`DELETE FROM forums`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "Not Found",
			id:   999,
			mock: func() {
				mock.ExpectExec(`DELETE FROM forums`).
					WithArgs(999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
		},
		{
			name: "Database Error",
			id:   1,
			mock: func() {
				mock.ExpectExec(`DELETE FROM forums`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			err := repo.Delete(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestForumsRepo_CreateMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewForumsRepo(db)

	testTime := time.Now()

	tests := []struct {
		name    string
		message models.Message
		mock    func()
		want    int
		wantErr bool
		errMsg  string
	}{
		{
			name: "Success",
			message: models.Message{
				ForumID:   1,
				Author:    "testuser",
				Content:   "Test message",
				CreatedAt: testTime,
			},
			mock: func() {
				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				mock.ExpectQuery(`INSERT INTO messages`).
					WithArgs(1, "testuser", "Test message", testTime).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			},
			want: 1,
		},
		{
			name: "Forum Not Found",
			message: models.Message{
				ForumID: 999,
			},
			mock: func() {
				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs(999).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
			},
			wantErr: true,
			errMsg:  "forum with ID 999 not found",
		},
		{
			name: "Database Error",
			message: models.Message{
				ForumID:   1,
				Author:    "testuser",
				Content:   "Test message",
				CreatedAt: testTime,
			},
			mock: func() {
				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				mock.ExpectQuery(`INSERT INTO messages`).
					WithArgs(1, "testuser", "Test message", testTime).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			got, err := repo.CreateMessage(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("CreateMessage() error = %v, wantErrMsg %v", err, tt.errMsg)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("CreateMessage() got = %v, want %v", got, tt.want)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestForumsRepo_GetMessages(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewForumsRepo(db)

	testTime := time.Now()

	tests := []struct {
		name    string
		forumID int
		mock    func()
		want    []models.Message
		wantErr bool
	}{
		{
			name:    "Success",
			forumID: 1,
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "forum_id", "author", "content", "created_at"}).
					AddRow(1, 1, "user1", "message 1", testTime).
					AddRow(2, 1, "user2", "message 2", testTime)
				mock.ExpectQuery(`SELECT id, forum_id, author, content, created_at`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			want: []models.Message{
				{ID: 1, ForumID: 1, Author: "user1", Content: "message 1", CreatedAt: testTime},
				{ID: 2, ForumID: 1, Author: "user2", Content: "message 2", CreatedAt: testTime},
			},
		},
		{
			name:    "Empty Result",
			forumID: 2,
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "forum_id", "author", "content", "created_at"})
				mock.ExpectQuery(`SELECT id, forum_id, author, content, created_at`).
					WithArgs(2).
					WillReturnRows(rows)
			},
			want:    []models.Message{},
			wantErr: false,
		},
		{
			name:    "Database Error",
			forumID: 1,
			mock: func() {
				mock.ExpectQuery(`SELECT id, forum_id, author, content, created_at`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			got, err := repo.GetMessages(tt.forumID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestForumsRepo_DeleteMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewForumsRepo(db)

	tests := []struct {
		name    string
		id      int
		mock    func()
		wantErr bool
	}{
		{
			name: "Success",
			id:   1,
			mock: func() {
				mock.ExpectExec(`DELETE FROM messages`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "Database Error",
			id:   1,
			mock: func() {
				mock.ExpectExec(`DELETE FROM messages`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			err := repo.DeleteMessage(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteMessage() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestForumsRepo_PutMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewForumsRepo(db)

	testTime := time.Now()

	tests := []struct {
		name           string
		messageID      int
		updatedContent string
		mock           func()
		want           *models.Message
		wantErr        bool
	}{
		{
			name:           "Success",
			messageID:      1,
			updatedContent: "updated content",
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "forum_id", "author", "content", "created_at"}).
					AddRow(1, 1, "testuser", "updated content", testTime)
				mock.ExpectQuery(`UPDATE messages`).
					WithArgs("updated content", 1).
					WillReturnRows(rows)
			},
			want: &models.Message{
				ID:        1,
				ForumID:   1,
				Author:    "testuser",
				Content:   "updated content",
				CreatedAt: testTime,
			},
		},
		{
			name:           "Not Found",
			messageID:      999,
			updatedContent: "updated content",
			mock: func() {
				mock.ExpectQuery(`UPDATE messages`).
					WithArgs("updated content", 999).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name:           "Database Error",
			messageID:      1,
			updatedContent: "updated content",
			mock: func() {
				mock.ExpectQuery(`UPDATE messages`).
					WithArgs("updated content", 1).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			got, err := repo.PutMessage(tt.messageID, tt.updatedContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("PutMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestForumsRepo_CreateGlobalMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewForumsRepo(db)

	testTime := time.Now()

	tests := []struct {
		name    string
		message models.GlobalMessage
		mock    func()
		want    int
		wantErr bool
	}{
		{
			name: "Success",
			message: models.GlobalMessage{
				Author:    "testuser",
				Content:   "Test message",
				CreatedAt: testTime,
			},
			mock: func() {
				rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
				mock.ExpectQuery(`INSERT INTO chat_messages`).
					WithArgs("testuser", "Test message", testTime).
					WillReturnRows(rows)
			},
			want: 1,
		},
		{
			name: "Database Error",
			message: models.GlobalMessage{
				Author:    "testuser",
				Content:   "Test message",
				CreatedAt: testTime,
			},
			mock: func() {
				mock.ExpectQuery(`INSERT INTO chat_messages`).
					WithArgs("testuser", "Test message", testTime).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			got, err := repo.CreateGlobalMessage(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateGlobalMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("CreateGlobalMessage() got = %v, want %v", got, tt.want)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestForumsRepo_GetGlobalMessages(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewForumsRepo(db)

	testTime := time.Now()

	tests := []struct {
		name    string
		limit   int
		mock    func()
		want    []models.GlobalMessage
		wantErr bool
	}{
		{
			name:  "Success",
			limit: 10,
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "author", "message", "created_at"}).
					AddRow(1, "user1", "message 1", testTime).
					AddRow(2, "user2", "message 2", testTime)
				mock.ExpectQuery(`SELECT id, author, message, created_at`).
					WithArgs(10).
					WillReturnRows(rows)
			},
			want: []models.GlobalMessage{
				{ID: 1, Author: "user1", Content: "message 1", CreatedAt: testTime},
				{ID: 2, Author: "user2", Content: "message 2", CreatedAt: testTime},
			},
		},
		{

			name:  "Empty Result",
			limit: 10,
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "author", "message", "created_at"})
				mock.ExpectQuery(`SELECT id, author, message, created_at`).
					WithArgs(10).
					WillReturnRows(rows)
			},
			want:    []models.GlobalMessage{},
			wantErr: false,
		},
		{
			name:  "Database Error",
			limit: 10,
			mock: func() {
				mock.ExpectQuery(`SELECT id, author, message, created_at`).
					WithArgs(10).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			got, err := repo.GetGlobalMessages(tt.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGlobalMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestForumsRepo_DeleteGlobalMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewForumsRepo(db)

	tests := []struct {
		name    string
		id      int
		mock    func()
		wantErr bool
	}{
		{
			name: "Success",
			id:   1,
			mock: func() {
				mock.ExpectExec(`DELETE FROM chat_messages`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "Database Error",
			id:   1,
			mock: func() {
				mock.ExpectExec(`DELETE FROM chat_messages`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			err := repo.DeleteGlobalMessage(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteGlobalMessage() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestForumsRepo_GetGlobalChatHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewForumsRepo(db)

	testTime := time.Now()

	tests := []struct {
		name    string
		limit   int
		mock    func()
		want    []models.GlobalMessage
		wantErr bool
	}{
		{
			name:  "Success",
			limit: 10,
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "author", "message", "created_at"}).
					AddRow(1, "user1", "message 1", testTime).
					AddRow(2, "user2", "message 2", testTime)
				mock.ExpectQuery(`SELECT id, author, message, created_at`).
					WithArgs(10).
					WillReturnRows(rows)
			},
			want: []models.GlobalMessage{
				{ID: 1, Author: "user1", Content: "message 1", CreatedAt: testTime},
				{ID: 2, Author: "user2", Content: "message 2", CreatedAt: testTime},
			},
		},

		{
			name:  "Empty Result",
			limit: 10,
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "author", "message", "created_at"})
				mock.ExpectQuery(`SELECT id, author, message, created_at`).
					WithArgs(10).
					WillReturnRows(rows)
			},
			want:    []models.GlobalMessage{},
			wantErr: false,
		},

		{
			name:  "Database Error",
			limit: 10,
			mock: func() {
				mock.ExpectQuery(`SELECT id, author, message, created_at`).
					WithArgs(10).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			got, err := repo.GetGlobalChatHistory(tt.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGlobalChatHistory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestForumsRepo_GetUserByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewForumsRepo(db)

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
				rows := sqlmock.NewRows([]string{"id", "username", "email", "created_at", "updated_at", "role"}).
					AddRow(1, "testuser", "test@example.com", testTime, testTime, "user")
				mock.ExpectQuery(`SELECT id, username, email, created_at, updated_at, role`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			want: &models.User{
				ID:        1,
				Username:  "testuser",
				Email:     "test@example.com",
				CreatedAt: testTime,
				UpdatedAt: testTime,
				Role:      "user",
			},
		},
		{
			name:   "Not Found",
			userID: 999,
			mock: func() {
				mock.ExpectQuery(`SELECT id, username, email, created_at, updated_at, role`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name:   "Database Error",
			userID: 1,
			mock: func() {
				mock.ExpectQuery(`SELECT id, username, email, created_at, updated_at, role`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
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

func TestForumsRepo_GetMessageByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewForumsRepo(db)

	testTime := time.Now()

	tests := []struct {
		name      string
		messageID int
		mock      func()
		want      *models.Message
		wantErr   bool
	}{
		{
			name:      "Success",
			messageID: 1,
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "forum_id", "author", "content", "created_at"}).
					AddRow(1, 1, "testuser", "test message", testTime)
				mock.ExpectQuery(`SELECT id, forum_id, author, content, created_at`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			want: &models.Message{
				ID:        1,
				ForumID:   1,
				Author:    "testuser",
				Content:   "test message",
				CreatedAt: testTime,
			},
		},
		{
			name:      "Not Found",
			messageID: 999,
			mock: func() {
				mock.ExpectQuery(`SELECT id, forum_id, author, content, created_at`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name:      "Database Error",
			messageID: 1,
			mock: func() {
				mock.ExpectQuery(`SELECT id, forum_id, author, content, created_at`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			got, err := repo.GetMessageByID(tt.messageID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMessageByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

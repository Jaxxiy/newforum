package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jaxxiy/newforum/forum_service/internal/models"
)

type ForumsRepository interface {
	GetAll() ([]models.Forum, error)
	GetByID(id int) (*models.Forum, error)
	Create(forum models.Forum) (int, error)
	Update(id int, forum models.Forum) error
	Delete(id int) error
	GetMessages(forumID int) ([]models.Message, error)
	CreateMessage(msg models.Message) (int, error)
	GetMessageByID(id int) (*models.Message, error)
	PutMessage(id int, content string) (*models.Message, error)
	DeleteMessage(id int) error
	CreateGlobalMessage(msg models.GlobalMessage) (int, error)
	GetGlobalChatHistory(limit int) ([]models.GlobalMessage, error)
	GetUserByID(id int) (*models.User, error)
}

type ForumsRepo struct {
	DB *sql.DB
}

func NewForumsRepo(db *sql.DB) *ForumsRepo {
	return &ForumsRepo{
		DB: db,
	}
}

func (r *ForumsRepo) Create(f models.Forum) (int, error) {
	var id int
	err := r.DB.QueryRow(`
        INSERT INTO forums (name, description, created_at)
        VALUES ($1, $2, DEFAULT)
        RETURNING id`,
		f.Title, f.Description).Scan(&id)
	return id, err
}

func (r *ForumsRepo) GetAll() ([]models.Forum, error) {
	rows, err := r.DB.Query(`SELECT id, name, description, created_at FROM forums`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	forums := []models.Forum{}
	for rows.Next() {
		var f models.Forum
		if err := rows.Scan(&f.ID, &f.Title, &f.Description, &f.CreatedAt); err != nil {
			return nil, err
		}
		forums = append(forums, f)
	}
	return forums, nil
}

func (r *ForumsRepo) GetByID(id int) (*models.Forum, error) {
	query := `SELECT id, name, description FROM forums WHERE id = $1`
	row := r.DB.QueryRow(query, id)

	var forum models.Forum
	err := row.Scan(&forum.ID, &forum.Title, &forum.Description)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("forum not found")
		}
		return nil, err
	}

	return &forum, nil
}

func (r *ForumsRepo) Update(id int, f models.Forum) error {
	result, err := r.DB.Exec(
		`UPDATE forums SET name = $1, description = $2 WHERE id = $3`,
		f.Title, f.Description, id,
	)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("no forum found with the given ID")
	}
	return nil
}

func (r *ForumsRepo) Delete(id int) error {
	result, err := r.DB.Exec(
		`DELETE FROM forums WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("no forum found with the given ID")
	}
	return nil
}

func (r *ForumsRepo) CreateMessage(msg models.Message) (int, error) {
	var exists bool
	fmt.Println(msg.ForumID)
	err := r.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM forums WHERE id = $1)", msg.ForumID).Scan(&exists)
	if err != nil {
		return 0, fmt.Errorf("forum check failed: %v", err)
	}
	if !exists {
		return 0, fmt.Errorf("forum with ID %d not found", msg.ForumID)
	}

	var id int
	err = r.DB.QueryRow(
		"INSERT INTO messages (forum_id, author, content, created_at) VALUES ($1, $2, $3, $4) RETURNING id",
		msg.ForumID, msg.Author, msg.Content, msg.CreatedAt,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("insert message failed: %v", err)
	}

	return id, nil
}

func (r *ForumsRepo) GetMessages(forumID int) ([]models.Message, error) {
	rows, err := r.DB.Query(`
		SELECT id, forum_id, author, content, created_at 
		FROM messages 
		WHERE forum_id = $1 
		ORDER BY created_at`, forumID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []models.Message{}
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ID, &m.ForumID, &m.Author, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func (r *ForumsRepo) DeleteMessage(id int) error {
	_, err := r.DB.Exec("DELETE FROM messages WHERE id = $1", id)
	return err
}

func (r *ForumsRepo) PutMessage(messageID int, updatedContent string) (*models.Message, error) {
	var updatedMessage models.Message

	err := r.DB.QueryRow(`
        UPDATE messages 
        SET content = $1
        WHERE id = $2
        RETURNING id, forum_id, author, content, created_at`,
		updatedContent,
		messageID,
	).Scan(
		&updatedMessage.ID,
		&updatedMessage.ForumID,
		&updatedMessage.Author,
		&updatedMessage.Content,
		&updatedMessage.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}

	return &updatedMessage, nil
}

func (r *ForumsRepo) CreateGlobalMessage(msg models.GlobalMessage) (int, error) {
	var id int
	err := r.DB.QueryRow(`
		INSERT INTO chat_messages (author, message, created_at) 
		VALUES ($1, $2, $3) 
		RETURNING id`,
		msg.Author, msg.Content, msg.CreatedAt,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("insert global message failed: %w", err)
	}

	return id, nil
}

func (r *ForumsRepo) GetGlobalMessages(limit int) ([]models.GlobalMessage, error) {
	rows, err := r.DB.Query(`
		SELECT id, author, message, created_at
		FROM chat_messages
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get global messages: %w", err)
	}
	defer rows.Close()

	messages := []models.GlobalMessage{}
	for rows.Next() {
		var m models.GlobalMessage
		if err := rows.Scan(&m.ID, &m.Author, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan global message: %w", err)
		}
		messages = append(messages, m)
	}

	return messages, nil
}

func (r *ForumsRepo) DeleteGlobalMessage(id int) error {
	_, err := r.DB.Exec("DELETE FROM chat_messages WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete global message: %w", err)
	}
	return nil
}

func (r *ForumsRepo) GetGlobalChatHistory(limit int) ([]models.GlobalMessage, error) {
	rows, err := r.DB.Query(`
        SELECT id, author, message, created_at 
        FROM chat_messages
        ORDER BY created_at ASC 
        LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := []models.GlobalMessage{}
	for rows.Next() {
		var msg models.GlobalMessage
		err := rows.Scan(&msg.ID, &msg.Author, &msg.Content, &msg.CreatedAt)
		if err != nil {
			return nil, err
		}
		history = append(history, msg)
	}

	return history, nil
}

func (r *ForumsRepo) GetUserByID(userID int) (*models.User, error) {
	query := `
        SELECT id, username, email, created_at, updated_at, role
        FROM users
        WHERE id = $1`

	user := &models.User{}
	err := r.DB.QueryRow(query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Role,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *ForumsRepo) GetMessageByID(messageID int) (*models.Message, error) {
	var m models.Message
	err := r.DB.QueryRow(
		"SELECT id, forum_id, author, content, created_at FROM messages WHERE id = $1",
		messageID,
	).Scan(&m.ID, &m.ForumID, &m.Author, &m.Content, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

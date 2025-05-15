package service

import (
	"errors"
	"fmt"

	"github.com/jaxxiy/newforum/forum_service/internal/models"
	"github.com/jaxxiy/newforum/forum_service/internal/repository"
)

var (
	ErrEmptyTitle       = errors.New("forum title cannot be empty")
	ErrEmptyDescription = errors.New("forum description cannot be empty")
	ErrTitleTooLong     = errors.New("forum title too long")
	ErrEmptyContent     = errors.New("message content cannot be empty")
	ErrEmptyAuthor      = errors.New("message author cannot be empty")
	ErrInvalidForumID   = errors.New("invalid forum ID")
	ErrInvalidUserID    = errors.New("invalid user ID")
	ErrContentTooLong   = errors.New("message content too long")
	ErrInvalidLimit     = errors.New("invalid limit for chat history")
)

const (
	MaxTitleLength   = 255
	MaxContentLength = 5000
)

type ForumService struct {
	repo repository.ForumsRepository
}

func NewForumService(repo repository.ForumsRepository) *ForumService {
	return &ForumService{
		repo: repo,
	}
}

func (s *ForumService) validateForum(forum models.Forum) error {
	if forum.Title == "" {
		return ErrEmptyTitle
	}
	if forum.Description == "" {
		return ErrEmptyDescription
	}
	if len(forum.Title) > MaxTitleLength {
		return ErrTitleTooLong
	}
	return nil
}

func (s *ForumService) validateMessage(message models.Message) error {
	if message.ForumID <= 0 {
		return ErrInvalidForumID
	}
	if message.Author == "" {
		return ErrEmptyAuthor
	}
	if message.Content == "" {
		return ErrEmptyContent
	}
	if len(message.Content) > MaxContentLength {
		return ErrContentTooLong
	}
	return nil
}

func (s *ForumService) validateGlobalMessage(message models.GlobalMessage) error {
	if message.Author == "" {
		return ErrEmptyAuthor
	}
	if message.Content == "" {
		return ErrEmptyContent
	}
	if len(message.Content) > MaxContentLength {
		return ErrContentTooLong
	}
	return nil
}

func (s *ForumService) GetAllForums() ([]models.Forum, error) {
	return s.repo.GetAll()
}

func (s *ForumService) GetForumByID(id int) (*models.Forum, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid forum ID: %d", id)
	}
	return s.repo.GetByID(id)
}

func (s *ForumService) CreateForum(forum models.Forum) (int, error) {
	if err := s.validateForum(forum); err != nil {
		return 0, err
	}
	return s.repo.Create(forum)
}

func (s *ForumService) UpdateForum(id int, forum models.Forum) error {
	if id <= 0 {
		return fmt.Errorf("invalid forum ID: %d", id)
	}
	if err := s.validateForum(forum); err != nil {
		return err
	}
	return s.repo.Update(id, forum)
}

func (s *ForumService) DeleteForum(id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid forum ID: %d", id)
	}
	return s.repo.Delete(id)
}

func (s *ForumService) GetMessages(forumID int) ([]models.Message, error) {
	if forumID <= 0 {
		return nil, fmt.Errorf("invalid forum ID: %d", forumID)
	}
	return s.repo.GetMessages(forumID)
}

func (s *ForumService) CreateMessage(message models.Message) (int, error) {
	if err := s.validateMessage(message); err != nil {
		return 0, err
	}
	return s.repo.CreateMessage(message)
}

func (s *ForumService) GetMessageByID(id int) (*models.Message, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid message ID: %d", id)
	}
	return s.repo.GetMessageByID(id)
}

func (s *ForumService) UpdateMessage(id int, content string) (*models.Message, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid message ID: %d", id)
	}
	if content == "" {
		return nil, ErrEmptyContent
	}
	if len(content) > MaxContentLength {
		return nil, ErrContentTooLong
	}
	return s.repo.PutMessage(id, content)
}

func (s *ForumService) DeleteMessage(id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid message ID: %d", id)
	}
	return s.repo.DeleteMessage(id)
}

func (s *ForumService) GetUserByID(id int) (*models.User, error) {
	if id <= 0 {
		return nil, ErrInvalidUserID
	}
	return s.repo.GetUserByID(id)
}

func (s *ForumService) CreateGlobalMessage(message models.GlobalMessage) (int, error) {
	if err := s.validateGlobalMessage(message); err != nil {
		return 0, err
	}
	return s.repo.CreateGlobalMessage(message)
}

func (s *ForumService) GetGlobalChatHistory(limit int) ([]models.GlobalMessage, error) {
	if limit <= 0 {
		return nil, ErrInvalidLimit
	}
	return s.repo.GetGlobalChatHistory(limit)
}

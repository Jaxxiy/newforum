package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/jaxxiy/newforum/core/logger"
	"github.com/jaxxiy/newforum/forum_service/internal/models"
	pb "github.com/jaxxiy/newforum/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var log = logger.GetLogger()

type authClient struct {
	client pb.AuthServiceClient
	conn   *grpc.ClientConn
}

func NewClient(authServiceAddr string) (AuthClient, error) {
	log.Info("Connecting to auth service", logger.String("address", authServiceAddr))

	conn, err := grpc.Dial(authServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service: %w", err)
	}

	return &authClient{
		client: pb.NewAuthServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *authClient) GetUserByID(ctx context.Context, userID int) (*models.User, error) {
	resp, err := c.client.GetUserByID(ctx, &pb.GetUserByIDRequest{
		UserId: int64(userID),
	})
	if err != nil {
		log.Error("Error getting user by ID",
			logger.Error(err),
			logger.Int("userID", userID))
		return nil, err
	}

	return &models.User{
		ID:        int(resp.User.Id),
		Username:  resp.User.Username,
		Email:     resp.User.Email,
		Role:      resp.User.Role,
		CreatedAt: time.Unix(resp.User.CreatedAt, 0),
		UpdatedAt: time.Unix(resp.User.UpdatedAt, 0),
	}, nil
}

func (c *authClient) GetUserByToken(ctx context.Context, token string) (*pb.UserResponse, error) {
	log.Debug("Sending GetUserByToken request", logger.String("token", token))

	return c.client.GetUserByToken(ctx, &pb.GetUserByTokenRequest{
		Token: token,
	})
}

func (c *authClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

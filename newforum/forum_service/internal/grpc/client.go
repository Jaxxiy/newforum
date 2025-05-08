package grpc

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/jaxxiy/newforum/core/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	client pb.AuthServiceClient
	conn   *grpc.ClientConn
}

func NewClient(authServiceAddr string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("Connecting to auth service at %s", authServiceAddr)

	conn, err := grpc.DialContext(ctx, authServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service: %v", err)
	}

	client := pb.NewAuthServiceClient(conn)
	return &Client{
		client: client,
		conn:   conn,
	}, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) GetUserByID(ctx context.Context, userID int) (*pb.UserResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("gRPC client is not initialized")
	}

	req := &pb.GetUserRequest{
		UserId: int32(userID),
	}

	resp, err := c.client.GetUserByID(ctx, req)
	if err != nil {
		log.Printf("Error getting user by ID: %v", err)
		return nil, err
	}
	return resp, nil
}

func (c *Client) GetUserByToken(ctx context.Context, token string) (*pb.UserResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("gRPC client is not initialized")
	}

	if token == "" {
		return nil, nil
	}

	req := &pb.GetUserByTokenRequest{
		Token: token,
	}

	log.Printf("Sending GetUserByToken request with token: %s", token)

	resp, err := c.client.GetUserByToken(ctx, req)
	if err != nil {
		log.Printf("Error getting user by token: %v", err)
		return nil, err
	}
	return resp, nil
}

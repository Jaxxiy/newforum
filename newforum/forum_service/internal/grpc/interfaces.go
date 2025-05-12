package grpc

import (
	"context"

	"github.com/jaxxiy/newforum/core/proto"
)

type AuthClient interface {
	GetUserByToken(ctx context.Context, token string) (*proto.UserResponse, error)
	Close() error
}

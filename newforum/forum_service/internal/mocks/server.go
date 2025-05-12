package mocks

import (
	"context"

	pb "github.com/jaxxiy/newforum/core/proto"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type AuthServiceClient struct {
	mock.Mock
}

func (m *AuthServiceClient) GetUserByToken(ctx context.Context, in *pb.GetUserByTokenRequest, opts ...grpc.CallOption) (*pb.UserResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pb.UserResponse), args.Error(1)
}

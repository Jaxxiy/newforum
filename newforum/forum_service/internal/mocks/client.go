package mocks

import (
	"context"

	"github.com/jaxxiy/newforum/core/proto"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MockAuthClient struct {
	mock.Mock
}

func (m *MockAuthClient) GetUserByToken(ctx context.Context, token string) (*proto.UserResponse, error) {
	args := m.Called(ctx, token)

	if args.Get(0) == nil {
		if err := args.Error(1); err != nil {
			if s, ok := status.FromError(err); ok {
				return nil, s.Err()
			}
			return nil, status.Error(codes.Internal, err.Error())
		}
		return nil, status.Error(codes.NotFound, "token not found")
	}

	if resp, ok := args.Get(0).(*proto.UserResponse); ok {
		return resp, args.Error(1)
	}
	return nil, status.Error(codes.Internal, "invalid response type")
}

func (m *MockAuthClient) SetupSuccess(token string, userID int32, role string) {
	m.On("GetUserByToken", mock.Anything, token).Return(
		&proto.UserResponse{
			Id:       userID,
			Username: "testuser",
			Role:     role,
		},
		nil,
	)
}

func (m *MockAuthClient) SetupError(token string, code codes.Code, message string) {
	m.On("GetUserByToken", mock.Anything, token).Return(
		nil,
		status.Error(code, message),
	)
}

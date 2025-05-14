package grpc

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/jaxxiy/newforum/auth_service/internal/service"
	pb "github.com/jaxxiy/newforum/core/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type Server struct {
	pb.UnimplementedAuthServiceServer
	authService *service.AuthService
	grpcServer  *grpc.Server
}

func NewServer(authService *service.AuthService) *Server {
	return &Server{
		authService: authService,
	}
}

func (s *Server) GetUserByID(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	user, err := s.authService.GetUserByID(int(req.UserId))
	if err != nil {
		log.Printf("Error getting user by ID: %v", err)
		return nil, err
	}

	return &pb.UserResponse{
		Id:       int32(user.ID),
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
	}, nil
}

func (s *Server) GetUserByToken(ctx context.Context, req *pb.GetUserByTokenRequest) (*pb.UserResponse, error) {
	log.Printf("Received GetUserByToken request with token: %s", req.Token)

	user, err := s.authService.ValidateToken(req.Token)
	if err != nil {
		log.Printf("Error validating token: %v", err)
		return nil, err
	}

	log.Printf("Successfully validated token for user: %s", user.Username)

	return &pb.UserResponse{
		Id:       int32(user.ID),
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
	}, nil
}

func StartGRPCServer(authService *service.AuthService, port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	server := NewServer(authService)

	keepaliveParams := keepalive.ServerParameters{
		MaxConnectionIdle:     5 * time.Minute,
		MaxConnectionAge:      10 * time.Minute,
		MaxConnectionAgeGrace: 5 * time.Second,
		Time:                  5 * time.Second,
		Timeout:               1 * time.Second,
	}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepaliveParams),
	)

	pb.RegisterAuthServiceServer(grpcServer, server)
	server.grpcServer = grpcServer

	log.Printf("Starting gRPC server on port %s", port)
	return grpcServer.Serve(lis)
}

func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}

package grpc

import (
	"context"
	"net"
	"time"

	"github.com/jaxxiy/newforum/auth_service/internal/service"
	"github.com/jaxxiy/newforum/core/logger"
	pb "github.com/jaxxiy/newforum/core/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var log = logger.GetLogger()

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
		log.Error("Error getting user by ID",
			logger.Error(err),
			logger.Int("userID", int(req.UserId)))
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
	log.Debug("Received GetUserByToken request", logger.String("token", req.Token))

	user, err := s.authService.ValidateToken(req.Token)
	if err != nil {
		log.Error("Error validating token",
			logger.Error(err),
			logger.String("token", req.Token))
		return nil, err
	}

	log.Debug("Successfully validated token",
		logger.String("username", user.Username),
		logger.String("role", user.Role))

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
		log.Error("Failed to start TCP listener",
			logger.Error(err),
			logger.String("port", port))
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

	return grpcServer.Serve(lis)
}

func (s *Server) Stop() {
	if s.grpcServer != nil {
		log.Info("Stopping gRPC server gracefully")
		s.grpcServer.GracefulStop()
	}
}

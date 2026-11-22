package grpcsrv

import (
	"context"
	"errors"
	"log"

	identityv1 "github.com/foodly/contracts/gen/go/identity/v1"
	"github.com/foodly/identity/internal/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	identityv1.UnimplementedIdentityServiceServer
	users *user.Service
}

func New(users *user.Service) *Server {
	return &Server{users: users}
}

func (s *Server) Register(ctx context.Context, req *identityv1.RegisterRequest) (*identityv1.RegisterResponse, error) {
	id, err := s.users.Register(ctx, req.GetEmail(), req.GetPassword(), req.GetName())
	if err == nil {
		return &identityv1.RegisterResponse{UserId: id.String()}, nil
	}
	if errors.Is(err, user.ErrInvalidInput) {
		return nil, status.Error(codes.InvalidArgument, "invalid email, password or name")
	}
	if errors.Is(err, user.ErrEmailTaken) {
		return nil, status.Error(codes.AlreadyExists, "email already registered")
	}
	log.Printf("register: %v", err)
	return nil, status.Error(codes.Internal, "register failed")
}

func (s *Server) Login(ctx context.Context, req *identityv1.LoginRequest) (*identityv1.LoginResponse, error) {
	token, err := s.users.Login(ctx, req.GetEmail(), req.GetPassword())
	if err == nil {
		return &identityv1.LoginResponse{AccessToken: token}, nil
	}
	if errors.Is(err, user.ErrInvalidInput) {
		return nil, status.Error(codes.InvalidArgument, "invalid email or password")
	}
	if errors.Is(err, user.ErrUnauthenticated) {
		return nil, status.Error(codes.Unauthenticated, "invalid email or password")
	}
	log.Printf("login: %v", err)
	return nil, status.Error(codes.Internal, "login failed")
}

func (s *Server) ValidateToken(ctx context.Context, req *identityv1.ValidateTokenRequest) (*identityv1.ValidateTokenResponse, error) {
	id, ok := s.users.ValidateToken(req.GetAccessToken())
	if !ok {
		return &identityv1.ValidateTokenResponse{IsValid: false}, nil
	}
	return &identityv1.ValidateTokenResponse{
		UserId:  id.String(),
		IsValid: true,
	}, nil
}

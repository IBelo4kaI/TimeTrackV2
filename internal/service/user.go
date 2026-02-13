package service

import (
	"context"
	"timetrack/internal/adapter/grpc"
)

type UserService struct {
	grpcClient *grpc.Client
}

func NewUserService(client *grpc.Client) *UserService {
	return &UserService{
		grpcClient: client,
	}
}

func (s *UserService) GetUser(ctx context.Context, sessionToken string, entity string, action string) (*grpc.GetUsersResponse, error) {
	return s.grpcClient.GetUsers(ctx, &grpc.GetUsersRequest{
		PermissionRequest: &grpc.PermissionRequest{
			SessionToken: sessionToken,
			Service:      "time",
			Entity:       entity,
			Action:       action,
		},
		OnlyActive: true,
	})
}

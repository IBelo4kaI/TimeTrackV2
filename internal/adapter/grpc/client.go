package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	PermissionServiceClient
	UserServiceClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Client{
		PermissionServiceClient: NewPermissionServiceClient(conn),
		UserServiceClient:       NewUserServiceClient(conn),
	}, nil
}

func (c *Client) Validate(
	ctx context.Context,
	req *PermissionRequest,
) (*PermissionResponse, error) {
	return c.PermissionServiceClient.ValidatePermission(ctx, req)
}

func (c *Client) GetUsers(
	ctx context.Context,
	req *GetUsersRequest,
) (*GetUsersResponse, error) {
	return c.UserServiceClient.GetUsers(ctx, req)
}

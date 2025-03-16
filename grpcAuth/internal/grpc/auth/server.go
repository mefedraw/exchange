package auth

import (
	"context"
	val "github.com/asaskevich/govalidator"
	ssov1 "github.com/mefedraw/protos/gen/go/sso"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Auth interface {
	Login(
		ctx context.Context,
		username, password string,
		appId int) (token string, err error)
	RegisterNewUser(
		ctx context.Context,
		username, password string) (userID int64, err error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

type ServerApi struct {
	ssov1.UnimplementedAuthServer
	auth Auth
}

func Register(gRPC *grpc.Server, auth Auth) {
	ssov1.RegisterAuthServer(gRPC, &ServerApi{auth: auth})
}

const emptyValue = 0

func (s *ServerApi) Login(ctx context.Context, req *ssov1.LoginRequest) (*ssov1.LoginResponse, error) {
	if err := validateLoginRequest(req); err != nil {
		return nil, err
	}
	// TODO: implement login via auth service
	token, err := s.auth.Login(ctx, req.Email, req.Password, int(req.AppId))
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, err.Error())
	}

	return &ssov1.LoginResponse{
		Token: token,
	}, nil
}

func (s *ServerApi) Register(ctx context.Context, req *ssov1.RegisterRequest) (*ssov1.RegisterResponse, error) {

	if err := validateRegisterRequest(req); err != nil {
		return nil, err
	}

	userID, err := s.auth.RegisterNewUser(ctx, req.Email, req.Password)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &ssov1.RegisterResponse{
		UserId: userID,
	}, nil
}

func (s *ServerApi) IsAdmin(ctx context.Context, req *ssov1.IsAdminRequest) (*ssov1.IsAdminResponse, error) {
	if err := validateIsAdminRequest(req); err != nil {
		return nil, err
	}
	isAdmin, err := s.auth.IsAdmin(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, err.Error())
	}

	return &ssov1.IsAdminResponse{IsAdmin: isAdmin}, nil
}

func validateLoginRequest(req *ssov1.LoginRequest) error {
	if !val.IsEmail(req.Email) {
		return status.Errorf(codes.InvalidArgument, "Email not valid")
	}

	if req.Password == "" {
		return status.Errorf(codes.InvalidArgument, "Password is required")
	}

	if req.AppId == emptyValue {
		return status.Errorf(codes.InvalidArgument, "app_id is required")
	}

	return nil
}

func validateRegisterRequest(req *ssov1.RegisterRequest) error {
	if !val.IsEmail(req.Email) {
		return status.Errorf(codes.InvalidArgument, "Email not valid")
	}
	if req.Password == "" {
		return status.Errorf(codes.InvalidArgument, "Password is required")
	}

	return nil
}

func validateIsAdminRequest(req *ssov1.IsAdminRequest) error {
	if req.GetUserId() == emptyValue {
		return status.Errorf(codes.InvalidArgument, "UserId is required")
	}

	return nil
}

package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"golang.org/x/crypto/bcrypt"
)

const (
	passwordCost = 12
	tokenType    = "Bearer"
	decoyHash    = "$2a$12$2CEKxbS3/T6w9he1Vj5BvOm38IE4VxDyGfT/OVvWKzd6GGlvw2cqe"
)

type TokenIssuer interface {
	Issue(userId uuid.UUID, email string) (string, time.Time, error)
}

type Service struct {
	repository Repository
	tokens     TokenIssuer
}

func NewService(repository Repository, tokens TokenIssuer) *Service {
	return &Service{repository: repository, tokens: tokens}
}

func (service *Service) Login(ctx context.Context, request LoginRequest) (LoginResponse, error) {
	if errs := request.Validate(); len(errs) > 0 {
		return LoginResponse{}, apperror.Validation(errs)
	}

	user, err := service.repository.GetByEmail(ctx, NormalizeEmail(request.Email))

	if errors.Is(err, ErrUserNotFound) {
		_ = bcrypt.CompareHashAndPassword([]byte(decoyHash), []byte(request.Password))
		return LoginResponse{}, invalidCredentials()
	}

	if err != nil {
		return LoginResponse{}, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.StrPasswordUser), []byte(request.Password))

	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return LoginResponse{}, invalidCredentials()
	}

	if err != nil {
		return LoginResponse{}, fmt.Errorf("failed to compare password hash: %w", err)
	}

	signed, expiresAt, err := service.tokens.Issue(user.UidUser, user.StrEmailUser)

	if err != nil {
		return LoginResponse{}, err
	}

	return LoginResponse{
		Token:     signed,
		TokenType: tokenType,
		ExpiresAt: expiresAt,
		User:      toUserResponse(user),
	}, nil
}

func (service *Service) Me(ctx context.Context, userId uuid.UUID) (UserResponse, error) {
	user, err := service.repository.GetById(ctx, userId)

	if errors.Is(err, ErrUserNotFound) {
		return UserResponse{}, apperror.Unauthorized("INVALID_TOKEN", "The session user no longer exists")
	}

	if err != nil {
		return UserResponse{}, err
	}

	return toUserResponse(user), nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), passwordCost)

	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

func invalidCredentials() *apperror.AppError {
	return apperror.Unauthorized("INVALID_CREDENTIALS", "Email or password is incorrect")
}

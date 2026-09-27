package auth

import (
	"context"
	"strings"

	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

const (
	minAdminPasswordLength = 12
	maxNameLength          = 120
)

type UserWriter interface {
	Upsert(ctx context.Context, params db.UpsertUserParams) (db.AppUser, error)
}

type AdminAccount struct {
	Email    string
	Password string
	Name     string
}

func (account AdminAccount) Validate() map[string]string {
	errs := map[string]string{}

	if !IsValidEmail(NormalizeEmail(account.Email)) {
		errs["ADMIN_EMAIL"] = "Must be a valid email address."
	}

	if len(account.Password) < minAdminPasswordLength || len(account.Password) > maxPasswordLength {
		errs["ADMIN_PASSWORD"] = "Must be between 12 and 72 bytes long."
	}

	if name := strings.TrimSpace(account.Name); name == "" || len(name) > maxNameLength {
		errs["ADMIN_NAME"] = "Is required and must be 120 characters long at most."
	}

	return errs
}

func UpsertAdmin(ctx context.Context, writer UserWriter, account AdminAccount) (UserResponse, error) {
	if errs := account.Validate(); len(errs) > 0 {
		return UserResponse{}, apperror.Validation(errs)
	}

	hash, err := HashPassword(account.Password)

	if err != nil {
		return UserResponse{}, err
	}

	user, err := writer.Upsert(ctx, db.UpsertUserParams{
		Email:        NormalizeEmail(account.Email),
		PasswordHash: hash,
		Name:         strings.TrimSpace(account.Name),
	})

	if err != nil {
		return UserResponse{}, err
	}

	return toUserResponse(user), nil
}

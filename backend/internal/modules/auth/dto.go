package auth

import (
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

const (
	maxEmailLength    = 255
	maxPasswordLength = 72
)

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func IsValidEmail(email string) bool {
	address, err := mail.ParseAddress(email)

	return err == nil && address.Address == email && len(email) <= maxEmailLength
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (request LoginRequest) Validate() map[string]string {
	errs := map[string]string{}

	if !IsValidEmail(NormalizeEmail(request.Email)) {
		errs["email"] = "Email must be a valid address."
	}

	if request.Password == "" || len(request.Password) > maxPasswordLength {
		errs["password"] = "Password is required and must be 72 bytes long at most."
	}

	return errs
}

type UserResponse struct {
	Id    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
}

type LoginResponse struct {
	Token     string       `json:"token"`
	TokenType string       `json:"tokenType"`
	ExpiresAt time.Time    `json:"expiresAt"`
	User      UserResponse `json:"user"`
}

func toUserResponse(user db.AppUser) UserResponse {
	return UserResponse{
		Id:    user.UidUser,
		Email: user.StrEmailUser,
		Name:  user.StrNameUser,
	}
}

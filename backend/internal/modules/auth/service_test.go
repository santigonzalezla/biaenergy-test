package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
	"golang.org/x/crypto/bcrypt"
)

const testPassword = "correct-horse-battery"

var testExpiresAt = time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)

type fakeRepository struct {
	users         map[string]db.AppUser
	err           error
	lookedUpEmail string
}

func (fake *fakeRepository) GetByEmail(ctx context.Context, email string) (db.AppUser, error) {
	fake.lookedUpEmail = email

	if fake.err != nil {
		return db.AppUser{}, fake.err
	}

	user, ok := fake.users[email]
	if !ok {
		return db.AppUser{}, ErrUserNotFound
	}

	return user, nil
}

func (fake *fakeRepository) GetById(ctx context.Context, id uuid.UUID) (db.AppUser, error) {
	if fake.err != nil {
		return db.AppUser{}, fake.err
	}

	for _, user := range fake.users {
		if user.UidUser == id {
			return user, nil
		}
	}

	return db.AppUser{}, ErrUserNotFound
}

type fakeIssuer struct {
	err error
}

func (fake *fakeIssuer) Issue(userId uuid.UUID, email string) (string, time.Time, error) {
	if fake.err != nil {
		return "", time.Time{}, fake.err
	}

	return "token-for-" + email, testExpiresAt, nil
}

func newAdmin(t *testing.T) db.AppUser {
	t.Helper()

	hash, err := HashPassword(testPassword)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	return db.AppUser{
		UidUser:         uuid.New(),
		StrEmailUser:    "admin@bia.app",
		StrPasswordUser: hash,
		StrNameUser:     "Admin BIA",
	}
}

func assertAppError(t *testing.T, err error, status int, code string) {
	t.Helper()

	appErr := apperror.From(err)

	if appErr.Status != status || appErr.Code != code {
		t.Fatalf("error = %d %s (%v), want %d %s", appErr.Status, appErr.Code, err, status, code)
	}
}

func TestLogin(t *testing.T) {
	admin := newAdmin(t)

	t.Run("Valid credentials return a bearer token and the user", func(t *testing.T) {
		repository := &fakeRepository{users: map[string]db.AppUser{admin.StrEmailUser: admin}}
		service := NewService(repository, &fakeIssuer{})

		response, err := service.Login(context.Background(), LoginRequest{Email: "  Admin@BIA.app ", Password: testPassword})
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}

		if repository.lookedUpEmail != "admin@bia.app" {
			t.Fatalf("looked up %q, want the normalized email", repository.lookedUpEmail)
		}

		want := LoginResponse{
			Token:     "token-for-admin@bia.app",
			TokenType: "Bearer",
			ExpiresAt: testExpiresAt,
			User:      UserResponse{Id: admin.UidUser, Email: "admin@bia.app", Name: "Admin BIA"},
		}

		if response != want {
			t.Fatalf("response = %+v, want %+v", response, want)
		}
	})

	t.Run("Unknown email and wrong password give the same answer", func(t *testing.T) {
		service := NewService(&fakeRepository{users: map[string]db.AppUser{admin.StrEmailUser: admin}}, &fakeIssuer{})

		_, unknownErr := service.Login(context.Background(), LoginRequest{Email: "nobody@bia.app", Password: testPassword})
		_, wrongErr := service.Login(context.Background(), LoginRequest{Email: "admin@bia.app", Password: "wrong-password"})

		assertAppError(t, unknownErr, http.StatusUnauthorized, "INVALID_CREDENTIALS")
		assertAppError(t, wrongErr, http.StatusUnauthorized, "INVALID_CREDENTIALS")

		if unknownErr.Error() != wrongErr.Error() {
			t.Fatalf("messages differ: %q vs %q", unknownErr, wrongErr)
		}
	})

	t.Run("Invalid request is rejected before touching the database", func(t *testing.T) {
		repository := &fakeRepository{}

		_, err := NewService(repository, &fakeIssuer{}).Login(context.Background(), LoginRequest{Email: "admin"})

		assertAppError(t, err, http.StatusBadRequest, "VALIDATION_ERROR")

		if repository.lookedUpEmail != "" {
			t.Fatalf("repository was queried with %q", repository.lookedUpEmail)
		}
	})

	t.Run("Database error becomes 500", func(t *testing.T) {
		service := NewService(&fakeRepository{err: errors.New("connection refused")}, &fakeIssuer{})

		_, err := service.Login(context.Background(), LoginRequest{Email: "admin@bia.app", Password: testPassword})

		assertAppError(t, err, http.StatusInternalServerError, "INTERNAL_ERROR")
	})

	t.Run("Corrupted stored hash becomes 500, not 401", func(t *testing.T) {
		corrupted := admin
		corrupted.StrPasswordUser = "not-a-bcrypt-hash"
		service := NewService(&fakeRepository{users: map[string]db.AppUser{corrupted.StrEmailUser: corrupted}}, &fakeIssuer{})

		_, err := service.Login(context.Background(), LoginRequest{Email: "admin@bia.app", Password: testPassword})

		assertAppError(t, err, http.StatusInternalServerError, "INTERNAL_ERROR")
	})

	t.Run("Token signing error becomes 500", func(t *testing.T) {
		service := NewService(
			&fakeRepository{users: map[string]db.AppUser{admin.StrEmailUser: admin}},
			&fakeIssuer{err: errors.New("failed to sign token")},
		)

		_, err := service.Login(context.Background(), LoginRequest{Email: "admin@bia.app", Password: testPassword})

		assertAppError(t, err, http.StatusInternalServerError, "INTERNAL_ERROR")
	})
}

func TestDecoyHashCostsTheSameAsRealHashes(t *testing.T) {
	cost, err := bcrypt.Cost([]byte(decoyHash))
	if err != nil {
		t.Fatalf("decoyHash is not a bcrypt hash: %v", err)
	}

	if cost != passwordCost {
		t.Fatalf("decoyHash cost = %d, want %d so unknown emails take as long as wrong passwords", cost, passwordCost)
	}
}

func TestMe(t *testing.T) {
	admin := newAdmin(t)
	service := NewService(&fakeRepository{users: map[string]db.AppUser{admin.StrEmailUser: admin}}, &fakeIssuer{})

	t.Run("Returns the user without the password hash", func(t *testing.T) {
		response, err := service.Me(context.Background(), admin.UidUser)
		if err != nil {
			t.Fatalf("Me() error = %v", err)
		}

		if response != (UserResponse{Id: admin.UidUser, Email: "admin@bia.app", Name: "Admin BIA"}) {
			t.Fatalf("response = %+v", response)
		}
	})

	t.Run("Deleted user invalidates the session", func(t *testing.T) {
		_, err := service.Me(context.Background(), uuid.New())

		assertAppError(t, err, http.StatusUnauthorized, "INVALID_TOKEN")
	})
}

func TestHashPassword(t *testing.T) {
	t.Run("Produces a salted bcrypt hash that verifies", func(t *testing.T) {
		first, err := HashPassword(testPassword)
		if err != nil {
			t.Fatalf("HashPassword() error = %v", err)
		}

		second, _ := HashPassword(testPassword)

		if first == second {
			t.Fatal("two hashes of the same password are equal, want different salts")
		}

		if err := bcrypt.CompareHashAndPassword([]byte(first), []byte(testPassword)); err != nil {
			t.Fatalf("hash does not verify: %v", err)
		}
	})

	t.Run("Rejects passwords longer than 72 bytes", func(t *testing.T) {
		if _, err := HashPassword(strings.Repeat("a", 73)); err == nil {
			t.Fatal("HashPassword() error = nil, want error")
		}
	})
}

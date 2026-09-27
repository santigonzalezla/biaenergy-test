package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
	"golang.org/x/crypto/bcrypt"
)

type fakeWriter struct {
	saved db.UpsertUserParams
	calls int
	err   error
}

func (fake *fakeWriter) Upsert(ctx context.Context, params db.UpsertUserParams) (db.AppUser, error) {
	fake.calls++
	fake.saved = params

	if fake.err != nil {
		return db.AppUser{}, fake.err
	}

	return db.AppUser{
		UidUser:         uuid.New(),
		StrEmailUser:    params.Email,
		StrPasswordUser: params.PasswordHash,
		StrNameUser:     params.Name,
	}, nil
}

func TestUpsertAdmin(t *testing.T) {
	t.Run("Stores a normalized email and a bcrypt hash, never the password", func(t *testing.T) {
		writer := &fakeWriter{}
		account := AdminAccount{Email: " Admin@BIA.app ", Password: "a-long-admin-password", Name: " Admin BIA "}

		user, err := UpsertAdmin(context.Background(), writer, account)
		if err != nil {
			t.Fatalf("UpsertAdmin() error = %v", err)
		}

		if writer.saved.Email != "admin@bia.app" || writer.saved.Name != "Admin BIA" {
			t.Fatalf("saved = %+v, want normalized email and trimmed name", writer.saved)
		}

		if writer.saved.PasswordHash == account.Password {
			t.Fatal("the plain password was stored")
		}

		if err := bcrypt.CompareHashAndPassword([]byte(writer.saved.PasswordHash), []byte(account.Password)); err != nil {
			t.Fatalf("stored hash does not verify: %v", err)
		}

		if user.Email != "admin@bia.app" {
			t.Fatalf("user = %+v", user)
		}
	})

	t.Run("Invalid account is rejected before hashing or writing", func(t *testing.T) {
		writer := &fakeWriter{}

		_, err := UpsertAdmin(context.Background(), writer, AdminAccount{Email: "admin", Password: "short", Name: ""})

		assertAppError(t, err, http.StatusBadRequest, "VALIDATION_ERROR")

		if writer.calls != 0 {
			t.Fatal("the writer was called with an invalid account")
		}
	})

	t.Run("Database error is returned", func(t *testing.T) {
		writer := &fakeWriter{err: errors.New("connection refused")}

		_, err := UpsertAdmin(context.Background(), writer, AdminAccount{Email: "admin@bia.app", Password: "a-long-admin-password", Name: "Admin"})

		if err == nil || !strings.Contains(err.Error(), "connection refused") {
			t.Fatalf("UpsertAdmin() error = %v, want the database error", err)
		}
	})
}

func TestAdminAccountValidate(t *testing.T) {
	tests := []struct {
		name       string
		account    AdminAccount
		wantFields []string
	}{
		{
			name:    "Valid account",
			account: AdminAccount{Email: "admin@bia.app", Password: strings.Repeat("a", 12), Name: "Admin"},
		},
		{
			name:       "Password shorter than 12",
			account:    AdminAccount{Email: "admin@bia.app", Password: strings.Repeat("a", 11), Name: "Admin"},
			wantFields: []string{"ADMIN_PASSWORD"},
		},
		{
			name:       "Password longer than 72 bytes",
			account:    AdminAccount{Email: "admin@bia.app", Password: strings.Repeat("a", 73), Name: "Admin"},
			wantFields: []string{"ADMIN_PASSWORD"},
		},
		{
			name:       "Every field invalid",
			account:    AdminAccount{Email: "not-an-email", Password: "", Name: "   "},
			wantFields: []string{"ADMIN_EMAIL", "ADMIN_PASSWORD", "ADMIN_NAME"},
		},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			errs := tableTest.account.Validate()

			if len(errs) != len(tableTest.wantFields) {
				t.Fatalf("Validate() = %v, want errors on %v", errs, tableTest.wantFields)
			}

			for _, field := range tableTest.wantFields {
				if _, ok := errs[field]; !ok {
					t.Fatalf("Validate() = %v, missing error on %q", errs, field)
				}
			}
		})
	}
}

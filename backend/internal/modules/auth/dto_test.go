package auth

import (
	"strings"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Admin@BIA.app "); got != "admin@bia.app" {
		t.Fatalf("NormalizeEmail() = %q, want admin@bia.app", got)
	}
}

func TestLoginRequestValidate(t *testing.T) {
	tests := []struct {
		name       string
		request    LoginRequest
		wantFields []string
	}{
		{
			name:    "Valid credentials",
			request: LoginRequest{Email: "admin@bia.app", Password: "secret"},
		},
		{
			name:    "Email with spaces and uppercase is normalized",
			request: LoginRequest{Email: "  Admin@BIA.app ", Password: "secret"},
		},
		{
			name:       "Missing email and password",
			request:    LoginRequest{},
			wantFields: []string{"email", "password"},
		},
		{
			name:       "Email without domain",
			request:    LoginRequest{Email: "admin", Password: "secret"},
			wantFields: []string{"email"},
		},
		{
			name:       "Email with display name",
			request:    LoginRequest{Email: "Admin <admin@bia.app>", Password: "secret"},
			wantFields: []string{"email"},
		},
		{
			name:       "Password longer than bcrypt accepts",
			request:    LoginRequest{Email: "admin@bia.app", Password: strings.Repeat("a", 73)},
			wantFields: []string{"password"},
		},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			errs := tableTest.request.Validate()

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

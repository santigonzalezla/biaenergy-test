package token

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "a-test-secret-with-at-least-32-characters"

var issuedAt = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

func newTestIssuer(now time.Time) *Issuer {
	issuer := NewIssuer(testSecret, time.Hour)
	issuer.now = func() time.Time { return now }

	return issuer
}

func signWith(t *testing.T, method jwt.SigningMethod, key any, claims jwt.Claims) string {
	t.Helper()

	signed, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}

	return signed
}

func validClaims(subject string) registeredClaims {
	return registeredClaims{
		Email: "admin@bia.app",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuerName,
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(issuedAt.Add(time.Hour)),
		},
	}
}

func TestIssueAndVerify(t *testing.T) {
	userId := uuid.New()
	issuer := newTestIssuer(issuedAt)

	signed, expiresAt, err := issuer.Issue(userId, "admin@bia.app")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if !expiresAt.Equal(issuedAt.Add(time.Hour)) {
		t.Fatalf("expiresAt = %v, want issuedAt + ttl", expiresAt)
	}

	claims, err := issuer.Verify(signed)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if claims.UserId != userId || claims.Email != "admin@bia.app" || !claims.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("claims = %+v", claims)
	}
}

func TestIssueReturnsTheExpirationInUTC(t *testing.T) {
	bogota := time.FixedZone("America/Bogota", -5*60*60)

	_, expiresAt, err := newTestIssuer(issuedAt.In(bogota)).Issue(uuid.New(), "admin@bia.app")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if expiresAt.Location() != time.UTC {
		t.Fatalf("expiresAt location = %v, want UTC like every other API timestamp", expiresAt.Location())
	}
}

func TestIssueAlignsExpirationWithTheTokenPrecision(t *testing.T) {
	issuer := newTestIssuer(issuedAt.Add(750 * time.Millisecond))

	signed, expiresAt, err := issuer.Issue(uuid.New(), "admin@bia.app")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	claims, err := issuer.Verify(signed)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if !expiresAt.Equal(claims.ExpiresAt) {
		t.Fatalf("Issue() expiresAt = %v, token exp = %v", expiresAt, claims.ExpiresAt)
	}
}

func TestVerifyExpiration(t *testing.T) {
	signed, _, err := newTestIssuer(issuedAt).Issue(uuid.New(), "admin@bia.app")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	t.Run("Accepted inside the clock skew", func(t *testing.T) {
		_, err := newTestIssuer(issuedAt.Add(time.Hour + clockSkew/2)).Verify(signed)

		if err != nil {
			t.Fatalf("Verify() error = %v, want nil", err)
		}
	})

	t.Run("Rejected as expired after the clock skew", func(t *testing.T) {
		_, err := newTestIssuer(issuedAt.Add(time.Hour + clockSkew + time.Second)).Verify(signed)

		if !errors.Is(err, ErrExpiredToken) {
			t.Fatalf("Verify() error = %v, want ErrExpiredToken", err)
		}
	})
}

func TestVerifyRejectsInvalidTokens(t *testing.T) {
	subject := uuid.NewString()
	withoutExpiration := validClaims(subject)
	withoutExpiration.ExpiresAt = nil
	otherIssuer := validClaims(subject)
	otherIssuer.Issuer = "someone-else"

	valid := signWith(t, jwt.SigningMethodHS256, []byte(testSecret), validClaims(subject))
	parts := strings.Split(valid, ".")
	forgedPayload := signWith(t, jwt.SigningMethodHS256, []byte(testSecret), validClaims(uuid.NewString()))

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "Signed with another secret",
			token: signWith(t, jwt.SigningMethodHS256, []byte("another-secret-with-at-least-32-chars"), validClaims(subject)),
		},
		{
			name:  "Signed with an algorithm other than HS256",
			token: signWith(t, jwt.SigningMethodHS512, []byte(testSecret), validClaims(subject)),
		},
		{
			name:  "Unsigned token with alg none",
			token: signWith(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims(subject)),
		},
		{
			name:  "Payload swapped under a valid signature",
			token: parts[0] + "." + strings.Split(forgedPayload, ".")[1] + "." + parts[2],
		},
		{
			name:  "Issued by someone else",
			token: signWith(t, jwt.SigningMethodHS256, []byte(testSecret), otherIssuer),
		},
		{
			name:  "Without expiration",
			token: signWith(t, jwt.SigningMethodHS256, []byte(testSecret), withoutExpiration),
		},
		{
			name:  "Subject that is not a user id",
			token: signWith(t, jwt.SigningMethodHS256, []byte(testSecret), validClaims("admin")),
		},
		{
			name:  "Malformed string",
			token: "not-a-jwt",
		},
		{
			name:  "Empty string",
			token: "",
		},
	}

	issuer := newTestIssuer(issuedAt)

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			_, err := issuer.Verify(tableTest.token)

			if !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("Verify() error = %v, want ErrInvalidToken", err)
			}
		})
	}
}

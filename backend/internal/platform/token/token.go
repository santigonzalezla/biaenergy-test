package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	issuerName = "biaenergy-api"
	clockSkew  = 30 * time.Second
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

type Claims struct {
	UserId    uuid.UUID
	Email     string
	ExpiresAt time.Time
}

type registeredClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

type Issuer struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewIssuer(secret string, ttl time.Duration) *Issuer {
	return &Issuer{secret: []byte(secret), ttl: ttl, now: time.Now}
}

func (issuer *Issuer) Issue(userId uuid.UUID, email string) (string, time.Time, error) {
	issuedAt := issuer.now().UTC().Truncate(time.Second)
	expiresAt := issuedAt.Add(issuer.ttl)

	claims := registeredClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuerName,
			Subject:   userId.String(),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(issuer.secret)

	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return signed, expiresAt, nil
}

func (issuer *Issuer) Verify(raw string) (Claims, error) {
	var claims registeredClaims

	_, err := jwt.ParseWithClaims(raw, &claims, issuer.key,
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuerName),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(clockSkew),
		jwt.WithTimeFunc(issuer.now),
	)

	if errors.Is(err, jwt.ErrTokenExpired) {
		return Claims{}, ErrExpiredToken
	}

	if err != nil {
		return Claims{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	userId, err := uuid.Parse(claims.Subject)

	if err != nil {
		return Claims{}, fmt.Errorf("%w: subject is not a user id", ErrInvalidToken)
	}

	return Claims{UserId: userId, Email: claims.Email, ExpiresAt: claims.ExpiresAt.Time}, nil
}

func (issuer *Issuer) key(*jwt.Token) (any, error) {
	return issuer.secret, nil
}

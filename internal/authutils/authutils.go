package authutils

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const CMS_ISSUER = "shaunkeyscom"

type IdentityClaims struct {
	jwt.RegisteredClaims

	SshKeyFingerprint string `json:"ssh_key_fingerprint"`
}

func (claims *IdentityClaims) FillRegistered(subject uuid.UUID, duration time.Duration) {
	now := time.Now()
	claims.Issuer = CMS_ISSUER
	claims.Subject = subject.String()
	claims.Audience = []string{CMS_ISSUER}
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(duration))
	claims.NotBefore = jwt.NewNumericDate(now)
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.ID = rand.Text()
}

func SignToken(claims IdentityClaims, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func VerifyToken(tokenString string, secret string) (IdentityClaims, error) {
	var claims IdentityClaims
	_, err := jwt.ParseWithClaims(
		tokenString,
		&claims,
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("invalid algorithm")
			}
			return []byte(secret), nil
		},
		jwt.WithAudience(CMS_ISSUER),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithNotBeforeRequired(),
	)
	if err != nil {
		return IdentityClaims{}, err
	}
	return claims, nil
}

type claimsContextKey struct{}

func AuthenticateJwt(jwtSecret string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		authString := req.Header.Get("Authorization")
		if !strings.HasPrefix(authString, "Bearer ") {
			http.Error(resp, "Unauthenticated", http.StatusUnauthorized)
			return
		}
		tokenString := authString[7:]
		claims, err := VerifyToken(tokenString, jwtSecret)
		if err != nil {
			http.Error(resp, "Unauthenticated", http.StatusUnauthorized)
			return
		}

		// Insert claims into context
		ctx := context.WithValue(req.Context(), claimsContextKey{}, claims)
		req = req.WithContext(ctx)

		// Call handler
		handler.ServeHTTP(resp, req)
	})
}

func MustGetClaims(ctx context.Context) IdentityClaims {
	return ctx.Value(claimsContextKey{}).(IdentityClaims)
}

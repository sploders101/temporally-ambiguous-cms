package apiservices

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sploders101/personal-website/cmd/webserver/config"
	"github.com/sploders101/personal-website/cmd/webserver/dbapi"
	"github.com/sploders101/personal-website/internal/authutils"
	authv1 "github.com/sploders101/personal-website/internal/gen/proto/com/shaunkeys/auth/v1"
	"golang.org/x/crypto/ssh"
)

const (
	MAX_KEY_OFFERINGS = 15
	NONCE_SIZE        = 256
	CMS_PERMIT_ISSUER = "shaunkeyscom-auth-permit"
)

var ErrPermitExpired = connect.NewError(connect.CodePermissionDenied, errors.New("auth permit expired"))

type AuthService struct {
	config config.ServerConfig
	db     dbapi.Db
}

func NewAuthService(config config.ServerConfig, db dbapi.Db) AuthService {
	return AuthService{
		config: config,
		db:     db,
	}
}

func (auth AuthService) CheckSSHKey(ctx context.Context, req *authv1.CheckSSHKeyRequest) (*authv1.CheckSSHKeyResponse, error) {
	_, err := verifyPermit(req.GetPermit(), auth.config.Secrets.AuthPermitSecret)
	if err != nil {
		slog.Error("Failed permit", "error", err)
		return nil, ErrPermitExpired
	}

	_, err = auth.db.Query().GetUserBySshKey(ctx, req.GetFingerprint())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Invalid key. Return normal response
			return authv1.CheckSSHKeyResponse_builder{
				Recognized: false,
			}.Build(), nil
		}
		slog.Error("Error getting user by SSH key", "error", err)
		return nil, ErrAmbiguousInternal
	}
	return authv1.CheckSSHKeyResponse_builder{
		Recognized: true,
	}.Build(), nil
}

type AuthPermitClaims struct {
	jwt.RegisteredClaims
}

func (auth AuthService) GetAuthPermit(ctx context.Context, req *authv1.GetAuthPermitRequest) (*authv1.GetAuthPermitResponse, error) {
	now := time.Now()
	exp := now.Add(2 * time.Minute)
	claims := AuthPermitClaims{
		ID: rand.Text(),
		Issuer: "shaunkeyscom-cms-auth",
		Audience: jwt.ClaimStrings{CMS_PERMIT_ISSUER},
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(exp),
		IssuedAt: jwt.NewNumericDate(now),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(auth.config.Secrets.AuthPermitSecret))
	if err != nil {
		slog.Error("Failed to sign auth permit", "error", err)
		return nil, ErrAmbiguousInternal
	}
	return authv1.GetAuthPermitResponse_builder{
		Permit: tokenString,
	}.Build(), nil
}

func (auth AuthService) ExchangeSSHPermit(ctx context.Context, req *authv1.ExchangeSSHPermitRequest) (*authv1.ExchangeSSHPermitResponse, error) {
	permit := req.GetPermit()
	fingerprint := req.GetFingerprint()
	format := req.GetFormat()
	signature := req.GetSignature()

	// Step 1: Validate Permit
	_, err := verifyPermit(permit, auth.config.Secrets.AuthPermitSecret)
	if err != nil {
		return nil, ErrPermitExpired
	}

	// Step 2: Get user info
	userBundle, err := auth.db.Query().GetUserBySshKey(ctx, fingerprint)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Expected. Send rejection
			return nil, ErrAuthenticationFailed
		}
		slog.Error("Failed to get user by SSH key", "error", err)
		return nil, ErrAmbiguousInternal
	}

	// Step 3: Validate SSH signature
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(userBundle.UsersSshKey.PublicKey))
	if err != nil {
		return nil, ErrAmbiguousInternal
	}
	if err := key.Verify([]byte(permit), &ssh.Signature{
		Format: format,
		Blob:   signature,
	}); err != nil {
		return nil, ErrAuthenticationFailed
	}

	// User is authenticated. Log the attempt and send them a key.
	if err := auth.db.Query().LogSSHAuthentication(ctx, fingerprint); err != nil {
		slog.Error("Failed to update SSH key last_used", "error", err)
		return nil, ErrAmbiguousInternal
	}

	// Step 4: Return user credential
	claims := authutils.IdentityClaims{
		SshKeyFingerprint: fingerprint,
	}
	claims.FillRegistered(userBundle.User.ID, 15*time.Minute)
	tokenString, err := authutils.SignToken(claims, auth.config.Secrets.JwtSecret)
	if err != nil {
		slog.Error("Failed to sign JWT", "error", err)
		return nil, ErrAmbiguousInternal
	}
	return authv1.ExchangeSSHPermitResponse_builder{
		Token: tokenString,
	}.Build(), nil
}

func verifyPermit(tokenString string, secret string) (AuthPermitClaims, error) {
	var claims AuthPermitClaims
	_, err := jwt.ParseWithClaims(
		tokenString,
		&claims,
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("invalid algorithm")
			}
			return []byte(secret), nil
		},
		jwt.WithAudience(CMS_PERMIT_ISSUER),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithNotBeforeRequired(),
	)
	if err != nil {
		return AuthPermitClaims{}, err
	}
	return claims, nil
}

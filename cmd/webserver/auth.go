package main

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/sploders101/personal-website/cmd/webserver/config"
	"github.com/sploders101/personal-website/cmd/webserver/dbapi"
	queries "github.com/sploders101/personal-website/cmd/webserver/dbapi/gen"
	"github.com/sploders101/personal-website/cmd/webserver/helpers"
	"github.com/sploders101/personal-website/cmd/webserver/userdata"
	"github.com/sploders101/personal-website/internal/env"
	"golang.org/x/crypto/ssh"
)

const (
	argon2Time    = 2
	argon2Memory  = 19 * 1024
	argon2Threads = 1
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

var argon2Params = &argon2id.Params{
	Memory:      argon2Memory,
	Iterations:  argon2Time,
	Parallelism: argon2Threads,
	SaltLength:  argon2SaltLen,
	KeyLength:   argon2KeyLen,
}

// dummyHash is verified against when a username does not exist, so that
// rejected logins take a similar amount of time whether or not the username
// exists. This mitigates user enumeration through response timing.
var dummyHash = mustHashPassword("shaunkeyscom-local-login-dummy")

func mustHashPassword(password string) string {
	encoded, err := hashPassword(password)
	if err != nil {
		panic(fmt.Sprintf("Failed to hash dummy password: %v", err))
	}
	return encoded
}

func serveLocalLogin(cfg config.ServerConfig, db dbapi.Db) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		if !cfg.Authentication.Local.Enabled {
			http.NotFound(resp, req)
			return
		}
		if err := req.ParseForm(); err != nil {
			slog.Error("Failed to parse local login form", "error", err)
			http.Error(resp, "Bad Request", http.StatusBadRequest)
			return
		}
		username := req.Form.Get("username")
		password := req.Form.Get("password")
		if username == "" || password == "" {
			http.Error(resp, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		cred, err := db.Query().GetUserByUsername(ctx, username)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Equalize response time with a real verification so that
				// username existence cannot be inferred from timing.
				_, _ = verifyPassword(password, sql.NullString{Valid: true, String: dummyHash})
				http.Error(resp, "Invalid credentials", http.StatusUnauthorized)
				return
			}
			slog.Error("Failed to get user by username", "username", username, "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		matched, err := verifyPassword(password, cred.PasswordHash)
		if err != nil {
			slog.Error("Failed to verify password", "username", cred.Username, "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if !matched {
			http.Error(resp, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		token, err := generateSessionToken()
		if err != nil {
			slog.Error("Failed to generate session token", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		expiration := time.Now().Add(24 * time.Hour)
		tokenHash := sha256.Sum256([]byte(token))

		tx, err := db.Begin(ctx)
		if err != nil {
			slog.Error("Failed to open database transaction", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		if err := tx.Query().CreateUserSession(ctx, queries.CreateUserSessionParams{
			TokenHash: tokenHash[:],
			UserID:    cred.ID,
			Expires:   expiration,
		}); err != nil {
			slog.Error("Failed to create user session", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			slog.Error("Failed to commit transaction", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		http.SetCookie(resp, &http.Cookie{
			Name:     "shaunkeyscom-session",
			Value:    token,
			Expires:  expiration,
			HttpOnly: true,
			Secure:   !env.Devmode,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		})
		http.Redirect(resp, req, "/", http.StatusSeeOther)
	})
}

// hashPassword derives a password Argon2id hash and returns it in PHC string
// format (self-describing, so parameters can evolve over time).
func hashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2Params)
}

// verifyPassword checks a password against a stored PHC-format Argon2id hash
// in constant time. An empty stored hash never matches.
func verifyPassword(password string, encoded sql.NullString) (bool, error) {
	if !encoded.Valid || encoded.String == "" {
		return false, nil
	}
	return argon2id.ComparePasswordAndHash(password, encoded.String)
}

func serveLogout(db dbapi.Db) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		tx, err := db.Begin(req.Context())
		if err != nil {
			slog.Error("Failed to open database transaction", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()
		userSession := userdata.GetSessionInfo(req.Context())
		if err := tx.Query().DeleteUserSession(req.Context(), userSession.TokenHash); err != nil {
			slog.Error("Failed to delete user session", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if err := tx.Commit(); err != nil {
			slog.Error("Failed to commit transaction", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Delete cookie by updating it with a `0` expiration time
		http.SetCookie(resp, &http.Cookie{
			Name:     "shaunkeyscom-session",
			Value:    "",
			Expires:  time.Unix(0, 0),
			HttpOnly: true,
			Secure:   !env.Devmode,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		})

		http.Redirect(resp, req, "/", http.StatusFound)
	})
}

func editUserProfile(db dbapi.Db) http.Handler {
	return helpers.RequireLogin(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		userDetails := userdata.GetUserData(ctx)
		tx, err := db.Begin(ctx)
		if err != nil {
			slog.Error("Failed to open database transaction", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		if err := req.ParseForm(); err != nil {
			slog.Debug("Failed to parse form", "error", err)
			http.Error(resp, "Bad request format", http.StatusBadRequest)
			return
		}

		newUsername := req.Form.Get("username")
		newEmail := req.Form.Get("email")

		if err := tx.Query().UpdateUserInfo(ctx, queries.UpdateUserInfoParams{
			ID:       userDetails.ID,
			Username: newUsername,
			Email:    newEmail,
		}); err != nil {
			slog.Error("Failed to update user info", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			slog.Error("Failed to commit db transaction", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		http.Redirect(resp, req, "/profile/", http.StatusFound)
	}))
}

func changePassword(db dbapi.Db) http.Handler {
	return helpers.RequireLogin(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		userDetails := userdata.GetUserData(ctx)

		if err := req.ParseForm(); err != nil {
			slog.Debug("Unable to parse form", "error", err)
			http.Error(resp, "Bad request format", http.StatusBadRequest)
			return
		}

		oldpass := req.Form.Get("oldpass")
		newpass := req.Form.Get("newpass")
		if req.Form.Has("newpassconfirm") {
			if newpass != req.Form.Get("newpassconfirm") {
				http.Redirect(resp, req, "/profile/change_password/?error=Passwords+did+not_match.", http.StatusFound)
				return
			}
		}

		// Verify old password
		pass, err := verifyPassword(oldpass, userDetails.PasswordHash)
		if err != nil {
			slog.Error("Failed to validate password", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if !pass {
			http.Error(resp, "Invalid password", http.StatusUnauthorized)
			return
		}

		// Hash new password
		newHash, err := hashPassword(newpass)
		if err != nil {
			slog.Error("Failed to hash new password", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			slog.Error("Failed to start db transaction", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()
		if err := tx.Query().SetUserPassword(ctx, queries.SetUserPasswordParams{
			ID:           userDetails.ID,
			PasswordHash: sql.NullString{Valid: true, String: newHash},
		}); err != nil {
			slog.Error("Failed to commit db transaction", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if err := tx.Commit(); err != nil {
			slog.Error("Failed to commit db transaction", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}))
}

func addSSHKey(db dbapi.Db) http.Handler {
	return helpers.RequireLogin(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		userDetails := userdata.GetUserData(ctx)

		if err := req.ParseForm(); err != nil {
			slog.Debug("Unable to parse form", "error", err)
			http.Error(resp, "Bad request format", http.StatusBadRequest)
			return
		}

		name := req.Form.Get("name")
		newKey := req.Form.Get("sshkey")

		if newKey == "" {
			http.Redirect(resp, req, "/profile/", http.StatusFound)
			return
		}

		key, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(newKey))
		if err != nil {
			http.Error(resp, "Invalid key", http.StatusBadRequest)
			return
		}
		fingerprint := ssh.FingerprintSHA256(key)
		if name == "" {
			name = comment
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			slog.Error("Failed to open database transaction", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		if _, err := tx.Query().AddSSHKey(ctx, queries.AddSSHKeyParams{
			UserID:      userDetails.ID,
			Name:        name,
			PublicKey:   newKey,
			Fingerprint: fingerprint,
			ExpiresAt:   sql.NullTime{}, // TODO: Implement this
		}); err != nil {
			slog.Error("Failed to add SSH key", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			slog.Error("Failed to commit database transaction", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		http.Redirect(resp, req, "/profile/", http.StatusFound)
	}))
}

func removeSSHKey(db dbapi.Db) http.Handler {
	return helpers.RequireLogin(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		userDetails := userdata.GetUserData(ctx)
		if err := req.ParseForm(); err != nil {
			slog.Debug("Unable to parse form", "error", err)
			http.Error(resp, "Bad request format", http.StatusBadRequest)
			return
		}

		keyId := req.Form.Get("keyId")
		if keyId == "" {
			http.Error(resp, "Missing keyId", http.StatusBadRequest)
			return
		}
		keyIdInt, err := strconv.ParseInt(keyId, 10, 64)
		if err != nil {
			http.Error(resp, "Invalid keyId", http.StatusBadRequest)
			return
		}

		tx, err := db.Begin(req.Context())
		if err != nil {
			slog.Debug("Unable to open database transaction", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if err := tx.Query().DeleteUserScopedSSHKey(
			req.Context(),
			queries.DeleteUserScopedSSHKeyParams{
				ID:     keyIdInt,
				UserID: userDetails.ID,
			},
		); err != nil {
			slog.Debug("Unable to delete user SSH key", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if err := tx.Commit(); err != nil {
			slog.Debug("Failed to commit transaction", "error", err)
			http.Error(resp, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		http.Redirect(resp, req, "/profile/", http.StatusFound)
	}))
}

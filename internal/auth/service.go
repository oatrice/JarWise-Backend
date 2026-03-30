package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"jarwise-backend/internal/models"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	SessionCookieName = "jarwise_session"
	sessionTTL        = 30 * 24 * time.Hour
)

var ErrUnauthorized = errors.New("unauthorized")

type GoogleIdentity struct {
	Subject   string
	Email     string
	Name      string
	AvatarURL string
}

type GoogleTokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken, audience string) (*GoogleIdentity, error)
}

type Service struct {
	db            *sql.DB
	verifier      GoogleTokenVerifier
	clientID      string
	secureCookies bool
	now           func() time.Time
}

func NewService(db *sql.DB, verifier GoogleTokenVerifier, clientID string, secureCookies bool) *Service {
	return &Service{
		db:            db,
		verifier:      verifier,
		clientID:      clientID,
		secureCookies: secureCookies,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Service) AuthenticateWithGoogle(ctx context.Context, idToken string) (*models.User, string, error) {
	if s.clientID == "" {
		return nil, "", fmt.Errorf("google auth is not configured")
	}
	if s.verifier == nil {
		return nil, "", fmt.Errorf("google token verifier is not configured")
	}

	identity, err := s.verifier.VerifyIDToken(ctx, idToken, s.clientID)
	if err != nil {
		return nil, "", ErrUnauthorized
	}

	user, err := s.upsertUser(ctx, identity)
	if err != nil {
		return nil, "", err
	}

	rawToken, err := generateOpaqueToken()
	if err != nil {
		return nil, "", err
	}

	if err := s.createSession(ctx, user.ID, rawToken); err != nil {
		return nil, "", err
	}

	return user, rawToken, nil
}

func (s *Service) AuthenticateRequest(ctx context.Context, rawToken string) (*models.User, error) {
	if rawToken == "" {
		return nil, ErrUnauthorized
	}

	user, err := s.lookupSessionUser(ctx, rawToken)
	if err != nil {
		return nil, err
	}

	if err := s.refreshSession(ctx, rawToken); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}

	_, err := s.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE token_hash = ?`, hashToken(rawToken))
	return err
}

func (s *Service) SetSessionCookie(w http.ResponseWriter, rawToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    rawToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
		Expires:  s.now().Add(sessionTTL),
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

func (s *Service) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func (s *Service) upsertUser(ctx context.Context, identity *GoogleIdentity) (*models.User, error) {
	now := s.now()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, google_sub, email, name, avatar_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(google_sub) DO UPDATE SET
			email = excluded.email,
			name = excluded.name,
			avatar_url = excluded.avatar_url,
			updated_at = excluded.updated_at
	`, uuid.NewString(), identity.Subject, identity.Email, identity.Name, identity.AvatarURL, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert user: %w", err)
	}

	user := &models.User{}
	err = s.db.QueryRowContext(ctx,
		`SELECT id, google_sub, email, name, avatar_url, created_at, updated_at FROM users WHERE google_sub = ?`,
		identity.Subject,
	).Scan(&user.ID, &user.GoogleSub, &user.Email, &user.Name, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	return user, nil
}

func (s *Service) createSession(ctx context.Context, userID, rawToken string) error {
	now := s.now()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_sessions (id, user_id, token_hash, expires_at, created_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, uuid.NewString(), userID, hashToken(rawToken), now.Add(sessionTTL), now, now)
	return err
}

func (s *Service) lookupSessionUser(ctx context.Context, rawToken string) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.google_sub, u.email, u.name, u.avatar_url, u.created_at, u.updated_at
		FROM user_sessions s
		INNER JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > ?
	`, hashToken(rawToken), s.now()).Scan(
		&user.ID,
		&user.GoogleSub,
		&user.Email,
		&user.Name,
		&user.AvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}

	return user, nil
}

func (s *Service) refreshSession(ctx context.Context, rawToken string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE user_sessions
		SET expires_at = ?, last_seen_at = ?
		WHERE token_hash = ?
	`, s.now().Add(sessionTTL), s.now(), hashToken(rawToken))
	return err
}

func generateOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

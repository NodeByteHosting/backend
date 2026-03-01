package database

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Session represents a user session
type Session struct {
	ID           string
	SessionToken string
	UserID       string
	Expires      time.Time
	CreatedAt    time.Time
}

// CreateSession creates a new session in the database
func (db *DB) CreateSession(ctx context.Context, userID string, sessionToken string, expiresAt time.Time) (*Session, error) {
	session := &Session{
		ID:           uuid.New().String(),
		SessionToken: sessionToken,
		UserID:       userID,
		Expires:      expiresAt,
		CreatedAt:    time.Now(),
	}

	query := `
		INSERT INTO sessions (id, "sessionToken", "userId", expires, "createdAt")
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, "sessionToken", "userId", expires, "createdAt"
	`

	err := db.Pool.QueryRow(ctx, query,
		session.ID,
		session.SessionToken,
		session.UserID,
		session.Expires,
		session.CreatedAt,
	).Scan(
		&session.ID,
		&session.SessionToken,
		&session.UserID,
		&session.Expires,
		&session.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetSessionByToken retrieves a session by its token
func (db *DB) GetSessionByToken(ctx context.Context, sessionToken string) (*Session, error) {
	session := &Session{}

	query := `
		SELECT id, "sessionToken", "userId", expires, "createdAt"
		FROM sessions
		WHERE "sessionToken" = $1 AND expires > NOW()
	`

	err := db.Pool.QueryRow(ctx, query, sessionToken).Scan(
		&session.ID,
		&session.SessionToken,
		&session.UserID,
		&session.Expires,
		&session.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return session, nil
}

// DeleteSession deletes a session from the database
func (db *DB) DeleteSession(ctx context.Context, sessionToken string) error {
	query := `DELETE FROM sessions WHERE "sessionToken" = $1`
	_, err := db.Pool.Exec(ctx, query, sessionToken)
	return err
}

// DeleteUserSessions deletes all sessions for a user
func (db *DB) DeleteUserSessions(ctx context.Context, userID string) error {
	query := `DELETE FROM sessions WHERE "userId" = $1`
	_, err := db.Pool.Exec(ctx, query, userID)
	return err
}

// DeleteExpiredSessions deletes all expired sessions
func (db *DB) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	query := `DELETE FROM sessions WHERE expires < NOW()`
	result, err := db.Pool.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// UpdateSessionExpiry updates the expiry time of a session
func (db *DB) UpdateSessionExpiry(ctx context.Context, sessionToken string, newExpiry time.Time) error {
	query := `
		UPDATE sessions
		SET expires = $2
		WHERE session_token = $1
	`
	_, err := db.Pool.Exec(ctx, query, sessionToken, newExpiry)
	return err
}

package database

import (
	"context"
	"database/sql"
	"time"
)

// HytaleOAuthToken represents stored Hytale OAuth tokens
type HytaleOAuthToken struct {
	ID                string
	AccountID         string // Account/owner UUID from Hytale
	AccessToken       string
	RefreshToken      string
	AccessTokenExpiry time.Time
	ProfileUUID       sql.NullString // Selected game profile UUID
	Scope             string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	LastRefreshedAt   sql.NullTime
}

// HytaleGameSession represents a game session for a server
type HytaleGameSession struct {
	ID            string
	AccountID     string
	ProfileUUID   string
	SessionToken  string
	IdentityToken string
	ExpiresAt     time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// HytaleOAuthRepository handles Hytale OAuth token storage
type HytaleOAuthRepository struct {
	db *DB
}

// NewHytaleOAuthRepository creates a new Hytale OAuth repository
func NewHytaleOAuthRepository(db *DB) *HytaleOAuthRepository {
	return &HytaleOAuthRepository{db: db}
}

// SaveOAuthToken saves or updates an OAuth token
func (r *HytaleOAuthRepository) SaveOAuthToken(ctx context.Context, token *HytaleOAuthToken) error {
	now := time.Now()

	// Try to update first
	result, err := r.db.Pool.Exec(ctx,
		`UPDATE hytale_oauth_tokens 
		SET access_token = $2, refresh_token = $3, access_token_expiry = $4, 
		    scope = $5, updated_at = $6, last_refreshed_at = $7
		WHERE account_id = $1`,
		token.AccountID, token.AccessToken, token.RefreshToken,
		token.AccessTokenExpiry, token.Scope, now, sql.NullTime{Time: now, Valid: true},
	)

	if err != nil {
		return err
	}

	// If no rows updated, insert
	if result.RowsAffected() == 0 {
		_, err := r.db.Pool.Exec(ctx,
			`INSERT INTO hytale_oauth_tokens 
			(id, account_id, access_token, refresh_token, access_token_expiry, 
			 scope, created_at, updated_at, last_refreshed_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			generateUUID(), token.AccountID, token.AccessToken, token.RefreshToken,
			token.AccessTokenExpiry, token.Scope, now, now,
			sql.NullTime{Time: now, Valid: true},
		)
		return err
	}

	return nil
}

// GetOAuthToken retrieves an OAuth token by account ID
func (r *HytaleOAuthRepository) GetOAuthToken(ctx context.Context, accountID string) (*HytaleOAuthToken, error) {
	token := &HytaleOAuthToken{}

	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, account_id, access_token, refresh_token, access_token_expiry,
		 profile_uuid, scope, created_at, updated_at, last_refreshed_at
		FROM hytale_oauth_tokens
		WHERE account_id = $1`,
		accountID,
	).Scan(
		&token.ID, &token.AccountID, &token.AccessToken, &token.RefreshToken,
		&token.AccessTokenExpiry, &token.ProfileUUID, &token.Scope,
		&token.CreatedAt, &token.UpdatedAt, &token.LastRefreshedAt,
	)

	if err != nil {
		return nil, err
	}

	return token, nil
}

// UpdateProfileUUID updates the selected profile UUID
func (r *HytaleOAuthRepository) UpdateProfileUUID(ctx context.Context, accountID string, profileUUID string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE hytale_oauth_tokens 
		SET profile_uuid = $2, updated_at = $3
		WHERE account_id = $1`,
		accountID, profileUUID, time.Now(),
	)
	return err
}

// SaveGameSession saves or updates a game session
func (r *HytaleOAuthRepository) SaveGameSession(ctx context.Context, session *HytaleGameSession) error {
	now := time.Now()

	result, err := r.db.Pool.Exec(ctx,
		`UPDATE hytale_game_sessions 
		SET session_token = $3, identity_token = $4, expires_at = $5, updated_at = $6
		WHERE account_id = $1 AND profile_uuid = $2`,
		session.AccountID, session.ProfileUUID, session.SessionToken,
		session.IdentityToken, session.ExpiresAt, now,
	)

	if err != nil {
		return err
	}

	// If no rows updated, insert
	if result.RowsAffected() == 0 {
		_, err := r.db.Pool.Exec(ctx,
			`INSERT INTO hytale_game_sessions 
			(id, account_id, profile_uuid, session_token, identity_token, 
			 expires_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			generateUUID(), session.AccountID, session.ProfileUUID,
			session.SessionToken, session.IdentityToken, session.ExpiresAt, now, now,
		)
		return err
	}

	return nil
}

// GetGameSession retrieves a game session
func (r *HytaleOAuthRepository) GetGameSession(ctx context.Context, accountID, profileUUID string) (*HytaleGameSession, error) {
	session := &HytaleGameSession{}

	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, account_id, profile_uuid, session_token, identity_token, 
		 expires_at, created_at, updated_at
		FROM hytale_game_sessions
		WHERE account_id = $1 AND profile_uuid = $2`,
		accountID, profileUUID,
	).Scan(
		&session.ID, &session.AccountID, &session.ProfileUUID, &session.SessionToken,
		&session.IdentityToken, &session.ExpiresAt, &session.CreatedAt, &session.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return session, nil
}

// DeleteGameSession deletes a game session
func (r *HytaleOAuthRepository) DeleteGameSession(ctx context.Context, accountID, profileUUID string) error {
	_, err := r.db.Pool.Exec(ctx,
		`DELETE FROM hytale_game_sessions
		WHERE account_id = $1 AND profile_uuid = $2`,
		accountID, profileUUID,
	)
	return err
}

// GetAllOAuthTokens retrieves all OAuth tokens (for refresh scheduler)
func (r *HytaleOAuthRepository) GetAllOAuthTokens(ctx context.Context) ([]*HytaleOAuthToken, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, account_id, access_token, refresh_token, access_token_expiry,
		 profile_uuid, scope, created_at, updated_at, last_refreshed_at
		FROM hytale_oauth_tokens
		WHERE refresh_token IS NOT NULL AND refresh_token != ''
		ORDER BY updated_at ASC`,
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*HytaleOAuthToken
	for rows.Next() {
		token := &HytaleOAuthToken{}
		err := rows.Scan(
			&token.ID, &token.AccountID, &token.AccessToken, &token.RefreshToken,
			&token.AccessTokenExpiry, &token.ProfileUUID, &token.Scope,
			&token.CreatedAt, &token.UpdatedAt, &token.LastRefreshedAt,
		)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	return tokens, rows.Err()
}

// GetAllGameSessions retrieves all active game sessions (for refresh scheduler)
func (r *HytaleOAuthRepository) GetAllGameSessions(ctx context.Context) ([]*HytaleGameSession, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, account_id, profile_uuid, session_token, identity_token, 
		 expires_at, created_at, updated_at
		FROM hytale_game_sessions
		ORDER BY updated_at ASC`,
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*HytaleGameSession
	for rows.Next() {
		session := &HytaleGameSession{}
		err := rows.Scan(
			&session.ID, &session.AccountID, &session.ProfileUUID, &session.SessionToken,
			&session.IdentityToken, &session.ExpiresAt, &session.CreatedAt, &session.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

// UpdateGameSessionRefresh updates the last refresh time for a game session
func (r *HytaleOAuthRepository) UpdateGameSessionRefresh(ctx context.Context, accountID, profileUUID string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE hytale_game_sessions 
		SET updated_at = $3
		WHERE account_id = $1 AND profile_uuid = $2`,
		accountID, profileUUID, time.Now(),
	)
	return err
}

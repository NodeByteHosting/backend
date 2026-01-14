package database

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	VerificationTokenType  = "email_verification"
	PasswordResetTokenType = "password_reset"
	MagicLinkTokenType     = "magic_link"
	TokenExpiration        = 24 * time.Hour
	MagicLinkExpiration    = 30 * time.Minute
)

// VerificationToken represents an authentication token
type VerificationToken struct {
	ID        string
	UserID    string
	Token     string
	Type      string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// QueryUserByEmail retrieves a user by email address
func (db *DB) QueryUserByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}

	err := db.Pool.QueryRow(ctx,
		`SELECT 
			id, email, password, username, first_name, last_name, 
			roles, is_pterodactyl_admin, is_virtfusion_admin, is_system_admin,
			pterodactyl_id, email_verified, is_active, avatar_url,
			created_at, updated_at, last_login_at
		FROM users 
		WHERE email = $1`,
		email,
	).Scan(
		&user.ID, &user.Email, &user.Password, &user.Username,
		&user.FirstName, &user.LastName,
		&user.Roles, &user.IsPterodactylAdmin, &user.IsVirtfusionAdmin,
		&user.IsSystemAdmin, &user.PterodactylID, &user.EmailVerified,
		&user.IsActive, &user.AvatarURL,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// QueryUserByID retrieves a user by ID
func (db *DB) QueryUserByID(ctx context.Context, id string) (*User, error) {
	user := &User{}

	err := db.Pool.QueryRow(ctx,
		`SELECT 
			id, email, password, username, first_name, last_name, 
			roles, is_pterodactyl_admin, is_virtfusion_admin, is_system_admin,
			pterodactyl_id, email_verified, is_active, avatar_url,
			created_at, updated_at, last_login_at
		FROM users 
		WHERE id = $1`,
		id,
	).Scan(
		&user.ID, &user.Email, &user.Password, &user.Username,
		&user.FirstName, &user.LastName,
		&user.Roles, &user.IsPterodactylAdmin, &user.IsVirtfusionAdmin,
		&user.IsSystemAdmin, &user.PterodactylID, &user.EmailVerified,
		&user.IsActive, &user.AvatarURL,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// CreateUser creates a new user with a hashed password
func (db *DB) CreateUser(ctx context.Context, user *User, password string) (*User, error) {
	// Hash password with bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate UUID for user
	userID := generateUUID()
	now := time.Now()

	err = db.Pool.QueryRow(ctx,
		`INSERT INTO users 
		(id, email, password, username, first_name, last_name, roles, 
		is_pterodactyl_admin, is_virtfusion_admin, is_system_admin, 
		is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, email, username, first_name, last_name, roles`,
		userID, user.Email, string(hashedPassword), user.Username,
		user.FirstName, user.LastName, user.Roles,
		user.IsPterodactylAdmin, user.IsVirtfusionAdmin, user.IsSystemAdmin,
		true, now, now,
	).Scan(
		&user.ID, &user.Email, &user.Username,
		&user.FirstName, &user.LastName, &user.Roles,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	user.ID = userID
	user.CreatedAt = now
	user.UpdatedAt = now
	user.IsActive = true

	return user, nil
}

// VerifyPassword checks if the provided password matches the user's hashed password
func (u *User) VerifyPassword(password string) bool {
	if !u.Password.Valid {
		return false
	}

	err := bcrypt.CompareHashAndPassword(
		[]byte(u.Password.String),
		[]byte(password),
	)

	return err == nil
}

// StoreVerificationToken generates and stores an email verification token
func (db *DB) StoreVerificationToken(ctx context.Context, userID string, tokenType string, expiration time.Duration) (string, error) {
	// Generate random token
	token := generateRandomToken()
	hashedToken := hashToken(token)
	expiresAt := time.Now().Add(expiration)

	_, err := db.Pool.Exec(ctx,
		`INSERT INTO verification_tokens (user_id, token, type, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, type) DO UPDATE
		SET token = $2, expires_at = $4, created_at = $5`,
		userID, hashedToken, tokenType, expiresAt, time.Now(),
	)

	if err != nil {
		return "", fmt.Errorf("failed to store verification token: %w", err)
	}

	return token, nil
}

// VerifyEmailToken validates an email verification token and marks email as verified
func (db *DB) VerifyEmailToken(ctx context.Context, userID, token string) (bool, error) {
	hashedToken := hashToken(token)

	// Check token exists and is not expired
	var id string
	err := db.Pool.QueryRow(ctx,
		`SELECT id FROM verification_tokens 
		WHERE user_id = $1 AND token = $2 AND type = $3 AND expires_at > NOW()`,
		userID, hashedToken, VerificationTokenType,
	).Scan(&id)

	if err != nil {
		return false, err
	}

	// Mark email as verified and delete token
	_, err = db.Pool.Exec(ctx,
		`BEGIN;
		UPDATE users SET email_verified = NOW() WHERE id = $1;
		DELETE FROM verification_tokens WHERE user_id = $1 AND type = $2;
		COMMIT;`,
		userID, VerificationTokenType,
	)

	if err != nil {
		return false, fmt.Errorf("failed to verify email: %w", err)
	}

	return true, nil
}

// GetPasswordResetToken retrieves a password reset token
func (db *DB) GetPasswordResetToken(ctx context.Context, userID, token string) (bool, error) {
	hashedToken := hashToken(token)

	var id string
	err := db.Pool.QueryRow(ctx,
		`SELECT id FROM verification_tokens 
		WHERE user_id = $1 AND token = $2 AND type = $3 AND expires_at > NOW()`,
		userID, hashedToken, PasswordResetTokenType,
	).Scan(&id)

	if err != nil {
		return false, err
	}

	return true, nil
}

// ResetUserPassword validates reset token and updates password
func (db *DB) ResetUserPassword(ctx context.Context, userID, token, newPassword string) (bool, error) {
	// Validate token first
	valid, err := db.GetPasswordResetToken(ctx, userID, token)
	if err != nil || !valid {
		return false, fmt.Errorf("invalid or expired token")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return false, fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password and delete token in transaction
	_, err = db.Pool.Exec(ctx,
		`BEGIN;
		UPDATE users SET password = $1, updated_at = NOW() WHERE id = $2;
		DELETE FROM verification_tokens WHERE user_id = $2 AND type = $3;
		COMMIT;`,
		string(hashedPassword), userID, PasswordResetTokenType,
	)

	if err != nil {
		return false, fmt.Errorf("failed to reset password: %w", err)
	}

	return true, nil
}

// GetMagicLinkToken retrieves a magic link token
func (db *DB) GetMagicLinkToken(ctx context.Context, token string) (*VerificationToken, error) {
	hashedToken := hashToken(token)

	vt := &VerificationToken{}
	err := db.Pool.QueryRow(ctx,
		`SELECT user_id, token, type, expires_at, created_at 
		FROM verification_tokens 
		WHERE token = $1 AND type = $2 AND expires_at > NOW()`,
		hashedToken, MagicLinkTokenType,
	).Scan(&vt.UserID, &vt.Token, &vt.Type, &vt.ExpiresAt, &vt.CreatedAt)

	if err != nil {
		return nil, err
	}

	return vt, nil
}

// ConsumeMagicLinkToken validates magic link token and deletes it (one-time use)
func (db *DB) ConsumeMagicLinkToken(ctx context.Context, token string) (string, error) {
	hashedToken := hashToken(token)

	var userID string
	err := db.Pool.QueryRow(ctx,
		`DELETE FROM verification_tokens 
		WHERE token = $1 AND type = $2 AND expires_at > NOW()
		RETURNING user_id`,
		hashedToken, MagicLinkTokenType,
	).Scan(&userID)

	if err != nil {
		return "", fmt.Errorf("invalid or expired magic link")
	}

	return userID, nil
}

// UpdateLastLogin updates the user's last login timestamp
func (db *DB) UpdateLastLogin(ctx context.Context, userID string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE users SET last_login_at = NOW() WHERE id = $1`,
		userID,
	)
	return err
}

// Helper functions

// generateRandomToken creates a random token
func generateRandomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// hashToken hashes a token for storage
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// generateUUID generates a simple UUID
func generateUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16],
	)
}

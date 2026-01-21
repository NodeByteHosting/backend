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
			id, email, password, username, "firstName", "lastName", 
			roles, "isPterodactylAdmin", "isVirtfusionAdmin", "isSystemAdmin",
			"pterodactylId", "emailVerified", "isActive", "avatarUrl",
			"createdAt", "updatedAt", "lastLoginAt"
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
			id, email, password, username, "firstName", "lastName", 
			roles, "isPterodactylAdmin", "isVirtfusionAdmin", "isSystemAdmin",
			"pterodactylId", "emailVerified", "isActive", "avatarUrl",
			"createdAt", "updatedAt", "lastLoginAt"
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
		(id, email, password, username, "firstName", "lastName", roles, 
		"isPterodactylAdmin", "isVirtfusionAdmin", "isSystemAdmin", 
		"isActive", "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, email, username, "firstName", "lastName", roles`,
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
// Supports both $2a$ (Go bcrypt) and $2b$ (bcryptjs) hash formats
func (u *User) VerifyPassword(password string) bool {
	if !u.Password.Valid {
		return false
	}

	hash := u.Password.String

	// bcryptjs uses $2b$ prefix, but Go's bcrypt uses $2a$
	// They are compatible - we just need to normalize the prefix for Go's library
	// Replace $2b$ with $2a$ for compatibility
	if len(hash) > 4 && hash[:4] == "$2b$" {
		hash = "$2a$" + hash[4:]
	}

	err := bcrypt.CompareHashAndPassword(
		[]byte(hash),
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
		`INSERT INTO verification_tokens (identifier, token, type, expires)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (identifier, token) DO UPDATE
		SET expires = $4`,
		userID, hashedToken, tokenType, expiresAt,
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
	var tokenVal string
	err := db.Pool.QueryRow(ctx,
		`SELECT token FROM verification_tokens 
		WHERE identifier = $1 AND token = $2 AND type = $3 AND expires > NOW()`,
		userID, hashedToken, VerificationTokenType,
	).Scan(&tokenVal)

	if err != nil {
		return false, err
	}

	// Mark email as verified and delete token
	_, err = db.Pool.Exec(ctx,
		`BEGIN;
		UPDATE users SET "emailVerified" = NOW() WHERE id = $1;
		DELETE FROM verification_tokens WHERE identifier = $1 AND type = $2;
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

	var tokenVal string
	err := db.Pool.QueryRow(ctx,
		`SELECT token FROM verification_tokens 
		WHERE identifier = $1 AND token = $2 AND type = $3 AND expires > NOW()`,
		userID, hashedToken, PasswordResetTokenType,
	).Scan(&tokenVal)

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
		UPDATE users SET password = $1, "updatedAt" = NOW() WHERE id = $2;
		DELETE FROM verification_tokens WHERE identifier = $2 AND type = $3;
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
		`SELECT identifier, token, type, expires 
		FROM verification_tokens 
		WHERE token = $1 AND type = $2 AND expires > NOW()`,
		hashedToken, MagicLinkTokenType,
	).Scan(&vt.UserID, &vt.Token, &vt.Type, &vt.ExpiresAt)

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
		WHERE token = $1 AND type = $2 AND expires > NOW()
		RETURNING identifier`,
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
		`UPDATE users SET "lastLoginAt" = NOW() WHERE id = $1`,
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

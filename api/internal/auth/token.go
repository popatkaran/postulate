package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/popatkaran/postulate/api/internal/domain"
	"github.com/popatkaran/postulate/api/internal/repository"
)

const (
	jwtTTL          = 8 * time.Hour
	refreshTokenTTL = 30 * 24 * time.Hour
)

// TokenResponse holds the values returned to the caller after successful token issuance.
type TokenResponse struct {
	Token        string
	RefreshToken string
	ExpiresAt    time.Time
	Role         string
}

// TokenIssuer issues JWT session tokens and manages refresh token lifecycle.
type TokenIssuer struct {
	jwtSecret   []byte
	refreshRepo repository.RefreshTokenRepository
	userRepo    repository.UserRepository
}

// NewTokenIssuer constructs a TokenIssuer.
func NewTokenIssuer(jwtSecret string, refreshRepo repository.RefreshTokenRepository, userRepo repository.UserRepository) *TokenIssuer {
	return &TokenIssuer{jwtSecret: []byte(jwtSecret), refreshRepo: refreshRepo, userRepo: userRepo}
}

// IssueSessionToken creates a signed JWT (8h) and a single-use refresh token (30d).
// The raw refresh token is returned once and never stored — only its SHA-256 hash
// is persisted in refresh_tokens.
func (ti *TokenIssuer) IssueSessionToken(ctx context.Context, userID uuid.UUID, role string) (TokenResponse, error) {
	now := time.Now()
	expiresAt := now.Add(jwtTTL)

	claims := jwt.MapClaims{
		"sub":  userID.String(),
		"role": role,
		"iat":  now.Unix(),
		"exp":  expiresAt.Unix(),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(ti.jwtSecret)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("sign jwt: %w", err)
	}

	rawToken, tokenHash, err := generateRefreshToken()
	if err != nil {
		return TokenResponse{}, err
	}

	rt := &domain.RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(refreshTokenTTL),
	}
	if err := ti.refreshRepo.Create(ctx, rt); err != nil {
		return TokenResponse{}, fmt.Errorf("store refresh token: %w", err)
	}

	return TokenResponse{
		Token:        token,
		RefreshToken: rawToken,
		ExpiresAt:    expiresAt,
		Role:         role,
	}, nil
}

// RefreshSession validates a raw refresh token, rotates it, and issues a new JWT.
// The old refresh token record is deleted (single-use rotation).
// On any validation failure a generic error is returned — callers must not
// distinguish "not found" from "expired" in responses.
func (ti *TokenIssuer) RefreshSession(ctx context.Context, rawToken string) (TokenResponse, error) {
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	rt, err := ti.refreshRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("invalid refresh token")
	}
	if time.Now().After(rt.ExpiresAt) {
		return TokenResponse{}, fmt.Errorf("invalid refresh token")
	}
	if rt.UsedAt != nil {
		return TokenResponse{}, fmt.Errorf("invalid refresh token")
	}

	user, err := ti.userRepo.FindByID(ctx, rt.UserID)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("user not found")
	}

	// Delete old token before issuing new one (rotation).
	if err := ti.refreshRepo.DeleteByUserID(ctx, rt.UserID); err != nil {
		return TokenResponse{}, fmt.Errorf("rotate refresh token: %w", err)
	}

	return ti.IssueSessionToken(ctx, user.ID, string(user.Role))
}

// RevokeAllSessions deletes all refresh tokens for the given user.
// Outstanding JWTs remain valid until their natural 8h expiry — this is the
// accepted trade-off for a stateless JWT design.
func (ti *TokenIssuer) RevokeAllSessions(ctx context.Context, userID uuid.UUID) error {
	if err := ti.refreshRepo.DeleteByUserID(ctx, userID); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}
	return nil
}

// generateRefreshToken produces a cryptographically random 256-bit hex token
// and its SHA-256 hash.
func generateRefreshToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	raw = hex.EncodeToString(b)
	h := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(h[:])
	return raw, hash, nil
}

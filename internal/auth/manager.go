package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

// User represents an authenticated principal.
type User struct {
	Username string `json:"username"`
	Role     Role   `json:"role"`
}

// Config defines auth manager settings.
type Config struct {
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	Repository      Repository
}

// TokenPair bundles access and refresh tokens with expiry metadata.
type TokenPair struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

type refreshTokenState struct {
	username  string
	expiresAt time.Time
}

// Claims encodes JWT-style claims for access tokens.
type Claims struct {
	Subject   string `json:"sub"`
	Role      Role   `json:"role"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// Manager issues and validates HMAC-signed JWT tokens for API access.
type Manager struct {
	secret        []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	repo          Repository
	refreshTokens map[string]refreshTokenState
	mu            sync.RWMutex
}

// NewManager constructs a Manager with the provided configuration.
func NewManager(cfg Config) (*Manager, error) {
	secret := strings.TrimSpace(cfg.Secret)
	if secret == "" {
		return nil, ErrMissingSecret
	}

	accessTTL := cfg.AccessTokenTTL
	if accessTTL <= 0 {
		accessTTL = 15 * time.Minute
	}

	refreshTTL := cfg.RefreshTokenTTL
	if refreshTTL <= 0 {
		refreshTTL = 7 * 24 * time.Hour
	}

	if cfg.Repository == nil {
		return nil, fmt.Errorf("auth repository is required")
	}

	return &Manager{
		secret:        []byte(secret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		repo:          cfg.Repository,
		refreshTokens: make(map[string]refreshTokenState),
	}, nil
}

// Authenticate validates credentials and returns the associated user.
func (m *Manager) Authenticate(ctx context.Context, username, password string) (User, error) {
	record, err := m.repo.GetByUsername(ctx, username)
	if err != nil {
		if err == ErrUserNotFound {
			return User{}, ErrInvalidCredentials
		}
		return User{}, err
	}
	if err := bcrypt.CompareHashAndPassword(record.PasswordHash, []byte(password)); err != nil {
		return User{}, ErrInvalidCredentials
	}
	return User{Username: record.Username, Role: record.Role}, nil
}

// IssueTokens creates and stores a new access/refresh token pair for the user.
func (m *Manager) IssueTokens(user User) (TokenPair, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.issueTokensLocked(user)
}

// Refresh validates the refresh token and issues a new pair, invalidating the old token.
func (m *Manager) Refresh(ctx context.Context, refreshToken string) (TokenPair, User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, ok := m.refreshTokens[refreshToken]
	if !ok {
		return TokenPair{}, User{}, ErrInvalidToken
	}

	if time.Now().After(state.expiresAt) {
		delete(m.refreshTokens, refreshToken)
		return TokenPair{}, User{}, ErrRefreshTokenExpired
	}

	delete(m.refreshTokens, refreshToken)
	record, err := m.repo.GetByUsername(ctx, state.username)
	if err != nil {
		if err == ErrUserNotFound {
			return TokenPair{}, User{}, ErrUnauthorized
		}
		return TokenPair{}, User{}, err
	}

	tokens, err := m.issueTokensLocked(User{Username: record.Username, Role: record.Role})
	if err != nil {
		return TokenPair{}, User{}, err
	}
	return tokens, User{Username: record.Username, Role: record.Role}, nil
}

// RevokeRefreshToken deletes a refresh token, commonly used for logout flows.
func (m *Manager) RevokeRefreshToken(refreshToken string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.refreshTokens, refreshToken)
}

// ParseAccessToken validates and parses the supplied JWT, returning the associated user.
func (m *Manager) ParseAccessToken(ctx context.Context, token string) (User, error) {
	claims, err := m.parseAndValidate(token)
	if err != nil {
		return User{}, err
	}

	record, err := m.repo.GetByUsername(ctx, claims.Subject)
	if err != nil {
		if err == ErrUserNotFound {
			return User{}, ErrUnauthorized
		}
		return User{}, err
	}

	return User{Username: record.Username, Role: record.Role}, nil
}

func (m *Manager) issueTokensLocked(user User) (TokenPair, error) {
	accessToken, accessExpiry, err := m.createAccessToken(user)
	if err != nil {
		return TokenPair{}, err
	}

	refreshToken, refreshExpiry, err := m.createRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}

	m.refreshTokens[refreshToken] = refreshTokenState{
		username:  user.Username,
		expiresAt: refreshExpiry,
	}

	return TokenPair{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessExpiry,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshExpiry,
	}, nil
}

func (m *Manager) createAccessToken(user User) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(m.accessTTL)
	claims := Claims{
		Subject:   user.Username,
		Role:      user.Role,
		IssuedAt:  now.Unix(),
		ExpiresAt: expiresAt.Unix(),
	}

	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("encode token header: %w", err)
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("encode token payload: %w", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signature := m.sign(fmt.Sprintf("%s.%s", encodedHeader, encodedPayload))

	token := fmt.Sprintf("%s.%s.%s", encodedHeader, encodedPayload, signature)
	return token, expiresAt, nil
}

func (m *Manager) createRefreshToken() (string, time.Time, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", time.Time{}, fmt.Errorf("generate refresh token: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(bytes)
	return token, time.Now().Add(m.refreshTTL), nil
}

func (m *Manager) parseAndValidate(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}

	headerSegment, payloadSegment, signatureSegment := parts[0], parts[1], parts[2]

	if !m.verifySignature(headerSegment, payloadSegment, signatureSegment) {
		return Claims{}, ErrInvalidToken
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadSegment)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}

	if claims.Subject == "" || claims.ExpiresAt == 0 {
		return Claims{}, ErrInvalidToken
	}

	if time.Now().Unix() > claims.ExpiresAt {
		return Claims{}, ErrInvalidToken
	}

	if claims.Role == "" {
		claims.Role = RoleUser
	}

	return claims, nil
}

func (m *Manager) sign(message string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(message))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (m *Manager) verifySignature(headerSegment, payloadSegment, signatureSegment string) bool {
	expected := m.sign(fmt.Sprintf("%s.%s", headerSegment, payloadSegment))
	decodedProvided, err := base64.RawURLEncoding.DecodeString(signatureSegment)
	if err != nil {
		return false
	}
	decodedExpected, err := base64.RawURLEncoding.DecodeString(expected)
	if err != nil {
		return false
	}
	return hmac.Equal(decodedProvided, decodedExpected)
}

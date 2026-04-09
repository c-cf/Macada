package service

import (
	"context"
	"fmt"
	"time"

	"github.com/c-cf/macada/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles user registration, login, and JWT operations.
type AuthService struct {
	userRepo      domain.UserRepository
	workspaceRepo domain.WorkspaceRepository
	memberRepo    domain.WorkspaceMemberRepository
	jwtSecret     []byte
	tokenTTL      time.Duration
}

func NewAuthService(
	userRepo domain.UserRepository,
	workspaceRepo domain.WorkspaceRepository,
	memberRepo domain.WorkspaceMemberRepository,
	jwtSecret string,
) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		workspaceRepo: workspaceRepo,
		memberRepo:    memberRepo,
		jwtSecret:     []byte(jwtSecret),
		tokenTTL:      24 * time.Hour,
	}
}

// JWTClaims stores only user identity — workspace is selected per-request via header.
type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// WorkspaceInfo is a lightweight workspace summary returned in auth responses.
type WorkspaceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

// RegisterResult is returned after successful registration.
type RegisterResult struct {
	User       *domain.User    `json:"user"`
	Token      string          `json:"token"`
	ExpiresAt  time.Time       `json:"expires_at"`
	Workspaces []WorkspaceInfo `json:"workspaces"`
}

// LoginResult is returned after successful login.
type LoginResult struct {
	User       *domain.User    `json:"user"`
	Token      string          `json:"token"`
	ExpiresAt  time.Time       `json:"expires_at"`
	Workspaces []WorkspaceInfo `json:"workspaces"`
}

// Register creates a new user with a default workspace.
func (s *AuthService) Register(ctx context.Context, email, password, name string) (*RegisterResult, error) {
	existing, _ := s.userRepo.GetByEmail(ctx, email)
	if existing != nil {
		return nil, fmt.Errorf("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now().UTC()

	user := &domain.User{
		ID:           domain.NewUserID(),
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		CreatedAt:    now,
		UpdatedAt:    now,
		Type:         "user",
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	ws := &domain.Workspace{
		ID:          domain.NewWorkspaceID(),
		Name:        name + "'s Workspace",
		Description: "Default workspace",
		Metadata:    map[string]string{},
		CreatedAt:   now,
		UpdatedAt:   now,
		Type:        "workspace",
	}
	if err := s.workspaceRepo.Create(ctx, ws); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	member := &domain.WorkspaceMember{
		WorkspaceID: ws.ID,
		UserID:      user.ID,
		Role:        "owner",
		CreatedAt:   now,
	}
	if err := s.memberRepo.Add(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add workspace member: %w", err)
	}

	expiresAt := now.Add(s.tokenTTL)
	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &RegisterResult{
		User:      user,
		Token:     token,
		ExpiresAt: expiresAt,
		Workspaces: []WorkspaceInfo{
			{ID: ws.ID, Name: ws.Name, Role: "owner"},
		},
	}, nil
}

// Login authenticates a user and returns a JWT + workspace list.
func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	workspaces, err := s.listUserWorkspaces(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	expiresAt := time.Now().UTC().Add(s.tokenTTL)
	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &LoginResult{
		User:       user,
		Token:      token,
		ExpiresAt:  expiresAt,
		Workspaces: workspaces,
	}, nil
}

// ListUserWorkspaces returns all workspaces the user belongs to.
func (s *AuthService) ListUserWorkspaces(ctx context.Context, userID string) ([]WorkspaceInfo, error) {
	return s.listUserWorkspaces(ctx, userID)
}

// ValidateToken satisfies the middleware.TokenValidator interface.
// Returns only userID — workspace is determined by X-Workspace-Id header.
func (s *AuthService) ValidateToken(tokenString string) (userID string, err error) {
	claims, err := s.parseToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

func (s *AuthService) parseToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (s *AuthService) generateToken(user *domain.User) (string, error) {
	now := time.Now().UTC()
	claims := JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) listUserWorkspaces(ctx context.Context, userID string) ([]WorkspaceInfo, error) {
	members, err := s.memberRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	workspaces := make([]WorkspaceInfo, 0, len(members))
	for _, m := range members {
		ws, err := s.workspaceRepo.GetByID(ctx, m.WorkspaceID)
		if err != nil {
			continue
		}
		workspaces = append(workspaces, WorkspaceInfo{
			ID:   ws.ID,
			Name: ws.Name,
			Role: m.Role,
		})
	}
	return workspaces, nil
}

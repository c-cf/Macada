package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/cchu-code/managed-agents/internal/domain"
	"github.com/rs/zerolog/log"
)

// TokenValidator validates a JWT and returns the user ID.
type TokenValidator interface {
	ValidateToken(tokenString string) (userID string, err error)
}

// MembershipChecker verifies a user belongs to a workspace.
type MembershipChecker interface {
	ListByUser(ctx context.Context, userID string) ([]*domain.WorkspaceMember, error)
}

// Auth returns middleware that authenticates via JWT Bearer token OR X-Api-Key.
//
// JWT path: parse token → read X-Workspace-Id header → verify membership → inject context.
// API key path: resolve workspace from key → inject context (unchanged).
func Auth(apiKeyRepo domain.APIKeyRepository, jwtValidator TokenValidator, memberChecker MembershipChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// --- JWT Bearer path ---
			if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
				userID, err := jwtValidator.ValidateToken(tokenStr)
				if err != nil {
					writeAuthError(w, http.StatusUnauthorized, "invalid or expired token")
					return
				}

				// Require X-Workspace-Id header
				wsID := r.Header.Get("X-Workspace-Id")
				if wsID == "" {
					writeAuthError(w, http.StatusBadRequest, "X-Workspace-Id header is required for JWT authentication")
					return
				}

				// Verify user is a member of the workspace
				members, err := memberChecker.ListByUser(r.Context(), userID)
				if err != nil {
					writeAuthError(w, http.StatusInternalServerError, "failed to check workspace membership")
					return
				}
				found := false
				for _, m := range members {
					if m.WorkspaceID == wsID {
						found = true
						break
					}
				}
				if !found {
					writeAuthError(w, http.StatusForbidden, "you are not a member of this workspace")
					return
				}

				ctx := WithWorkspaceID(r.Context(), wsID)
				ctx = WithUserID(ctx, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// --- X-Api-Key path ---
			raw := r.Header.Get("X-Api-Key")
			if raw == "" {
				writeAuthError(w, http.StatusUnauthorized, "missing authentication (provide Authorization: Bearer <token> or X-Api-Key header)")
				return
			}

			hash := SHA256Hex(raw)
			key, err := apiKeyRepo.GetByHash(r.Context(), hash)
			if err != nil || key == nil {
				writeAuthError(w, http.StatusUnauthorized, "invalid API key")
				return
			}
			if key.RevokedAt != nil {
				writeAuthError(w, http.StatusUnauthorized, "API key has been revoked")
				return
			}
			if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
				writeAuthError(w, http.StatusUnauthorized, "API key has expired")
				return
			}

			ctx := WithWorkspaceID(r.Context(), key.WorkspaceID)
			ctx = WithAPIKeyID(ctx, key.ID)

			go func() {
				if err := apiKeyRepo.TouchLastUsed(context.Background(), key.ID); err != nil {
					log.Warn().Err(err).Str("key_id", key.ID).Msg("failed to update last_used_at")
				}
			}()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SHA256Hex computes hex-encoded SHA-256 of the input string.
func SHA256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"type":  "error",
		"error": map[string]string{"type": http.StatusText(status), "message": message},
	})
}

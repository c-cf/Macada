package handler

import (
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/c-cf/macada/internal/domain"
	"github.com/c-cf/macada/internal/infra/crypto"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// secretKeyRe restricts vault secret keys to safe alphanumeric identifiers.
var secretKeyRe = regexp.MustCompile(`^[A-Za-z0-9_\-\.]{1,255}$`)

type VaultHandler struct {
	repo      domain.VaultRepository
	encryptor *crypto.VaultEncryptor
}

func NewVaultHandler(repo domain.VaultRepository, encryptor *crypto.VaultEncryptor) *VaultHandler {
	return &VaultHandler{repo: repo, encryptor: encryptor}
}

// --- Vault CRUD ---

type createVaultRequest struct {
	DisplayName string            `json:"display_name"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

func (h *VaultHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createVaultRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "display_name is required")
		return
	}
	if len(req.DisplayName) > 255 {
		writeError(w, http.StatusBadRequest, "display_name must be 255 characters or fewer")
		return
	}
	if !validateMetadata(req.Metadata) {
		writeError(w, http.StatusBadRequest, "metadata must have at most 16 pairs, keys up to 64 chars, values up to 512 chars")
		return
	}

	now := time.Now().UTC()
	vault := &domain.Vault{
		ID:          domain.NewVaultID(),
		WorkspaceID: workspaceIDFromCtx(r),
		DisplayName: req.DisplayName,
		Metadata:    map[string]string{},
		CreatedAt:   now,
		UpdatedAt:   now,
		Type:        "vault",
	}
	if req.Metadata != nil {
		vault.Metadata = req.Metadata
	}

	if err := h.repo.Create(r.Context(), vault); err != nil {
		log.Error().Err(err).Msg("failed to create vault")
		writeError(w, http.StatusInternalServerError, "failed to create vault")
		return
	}

	writeJSON(w, http.StatusOK, vault)
}

func (h *VaultHandler) List(w http.ResponseWriter, r *http.Request) {
	lp := parseListParams(r)
	lp.WorkspaceID = workspaceIDFromCtx(r)
	params := domain.VaultListParams{ListParams: lp}

	vaults, nextPage, err := h.repo.List(r.Context(), params)
	if err != nil {
		log.Error().Err(err).Msg("failed to list vaults")
		writeError(w, http.StatusInternalServerError, "failed to list vaults")
		return
	}
	if vaults == nil {
		vaults = []*domain.Vault{}
	}

	writeJSON(w, http.StatusOK, domain.ListResponse[*domain.Vault]{
		Data:     vaults,
		NextPage: nextPage,
	})
}

func (h *VaultHandler) Retrieve(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "vault_id")
	vault, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "vault not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("vault_id", id).Msg("failed to retrieve vault")
		writeError(w, http.StatusInternalServerError, "failed to retrieve vault")
		return
	}
	if vault.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "vault not found")
		return
	}
	writeJSON(w, http.StatusOK, vault)
}

type updateVaultRequest struct {
	DisplayName *string            `json:"display_name,omitempty"`
	Metadata    map[string]*string `json:"metadata,omitempty"`
}

func (h *VaultHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "vault_id")
	wsID := workspaceIDFromCtx(r)

	vault, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "vault not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("vault_id", id).Msg("failed to retrieve vault for update")
		writeError(w, http.StatusInternalServerError, "failed to retrieve vault")
		return
	}
	if vault.WorkspaceID != wsID {
		writeError(w, http.StatusNotFound, "vault not found")
		return
	}

	var req updateVaultRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DisplayName != nil {
		if *req.DisplayName == "" || len(*req.DisplayName) > 255 {
			writeError(w, http.StatusBadRequest, "display_name must be 1-255 characters")
			return
		}
		vault.DisplayName = *req.DisplayName
	}

	// Metadata patch: set key to string to upsert, set to null to delete, omitted keys preserved.
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			if v == nil {
				delete(vault.Metadata, k)
			} else {
				vault.Metadata[k] = *v
			}
		}
		if !validateMetadata(vault.Metadata) {
			writeError(w, http.StatusBadRequest, "metadata must have at most 16 pairs, keys up to 64 chars, values up to 512 chars")
			return
		}
	}

	if err := h.repo.Update(r.Context(), vault, wsID); err != nil {
		if errors.Is(err, domain.ErrVaultNotFound) {
			writeError(w, http.StatusNotFound, "vault not found")
			return
		}
		log.Error().Err(err).Str("vault_id", id).Msg("failed to update vault")
		writeError(w, http.StatusInternalServerError, "failed to update vault")
		return
	}

	writeJSON(w, http.StatusOK, vault)
}

func (h *VaultHandler) Archive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "vault_id")
	wsID := workspaceIDFromCtx(r)

	vault, err := h.repo.Archive(r.Context(), id, wsID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "vault not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("vault_id", id).Msg("failed to archive vault")
		writeError(w, http.StatusInternalServerError, "failed to archive vault")
		return
	}
	writeJSON(w, http.StatusOK, vault)
}

func (h *VaultHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "vault_id")
	wsID := workspaceIDFromCtx(r)

	if err := h.repo.Delete(r.Context(), id, wsID); err != nil {
		if errors.Is(err, domain.ErrVaultNotFound) {
			writeError(w, http.StatusNotFound, "vault not found")
			return
		}
		log.Error().Err(err).Str("vault_id", id).Msg("failed to delete vault")
		writeError(w, http.StatusInternalServerError, "failed to delete vault")
		return
	}

	writeJSON(w, http.StatusOK, domain.DeletedVault{
		ID:   id,
		Type: "vault_deleted",
	})
}

// --- Secret operations ---

type setSecretRequest struct {
	Value string `json:"value"`
}

func (h *VaultHandler) SetSecret(w http.ResponseWriter, r *http.Request) {
	vaultID := chi.URLParam(r, "vault_id")
	wsID := workspaceIDFromCtx(r)

	key := chi.URLParam(r, "secret_key")
	if !secretKeyRe.MatchString(key) {
		writeError(w, http.StatusBadRequest, "secret key must match [A-Za-z0-9_\\-\\.]{1,255}")
		return
	}

	var req setSecretRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Value == "" {
		writeError(w, http.StatusBadRequest, "value is required")
		return
	}

	aad := crypto.SecretAAD(vaultID, key)
	encrypted, nonce, err := h.encryptor.Encrypt([]byte(req.Value), aad)
	if err != nil {
		log.Error().Err(err).Msg("failed to encrypt secret")
		writeError(w, http.StatusInternalServerError, "failed to encrypt secret")
		return
	}

	if err := h.repo.SetSecret(r.Context(), vaultID, wsID, key, encrypted, nonce); err != nil {
		if errors.Is(err, domain.ErrVaultNotFound) {
			writeError(w, http.StatusNotFound, "vault not found")
			return
		}
		log.Error().Err(err).Str("vault_id", vaultID).Msg("failed to store secret")
		writeError(w, http.StatusInternalServerError, "failed to store secret")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"type":     "vault_secret",
		"vault_id": vaultID,
		"key":      key,
	})
}

func (h *VaultHandler) DeleteSecret(w http.ResponseWriter, r *http.Request) {
	vaultID := chi.URLParam(r, "vault_id")
	wsID := workspaceIDFromCtx(r)
	key := chi.URLParam(r, "secret_key")

	if err := h.repo.DeleteSecret(r.Context(), vaultID, wsID, key); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "secret not found")
			return
		}
		log.Error().Err(err).Str("vault_id", vaultID).Str("key", key).Msg("failed to delete secret")
		writeError(w, http.StatusInternalServerError, "failed to delete secret")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"type":     "vault_secret_deleted",
		"vault_id": vaultID,
		"key":      key,
	})
}

func (h *VaultHandler) ListSecrets(w http.ResponseWriter, r *http.Request) {
	vaultID := chi.URLParam(r, "vault_id")
	wsID := workspaceIDFromCtx(r)

	secrets, err := h.repo.ListSecrets(r.Context(), vaultID, wsID)
	if err != nil {
		log.Error().Err(err).Str("vault_id", vaultID).Msg("failed to list secrets")
		writeError(w, http.StatusInternalServerError, "failed to list secrets")
		return
	}
	if secrets == nil {
		secrets = []*domain.VaultSecret{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": secrets,
	})
}

// --- helpers ---

// validateMetadata checks metadata constraints: max 16 pairs, keys ≤64 chars, values ≤512 chars.
func validateMetadata(m map[string]string) bool {
	if len(m) > 16 {
		return false
	}
	for k, v := range m {
		if len(k) > 64 || len(v) > 512 {
			return false
		}
	}
	return true
}

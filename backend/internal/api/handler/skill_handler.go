package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cchu-code/managed-agents/internal/domain"
	"github.com/cchu-code/managed-agents/internal/service"
	"github.com/go-chi/chi/v5"
)

type SkillHandler struct {
	repo domain.SkillRepository
}

func NewSkillHandler(repo domain.SkillRepository) *SkillHandler {
	return &SkillHandler{repo: repo}
}

type createSkillRequest struct {
	Content string `json:"content"` // raw SKILL.md text
}

// Create registers a new skill.
//
// Content-Type: application/json  → body contains {"content": "<SKILL.md text>"}
// Content-Type: multipart/form-data → "file" field contains a .zip archive
func (h *SkillHandler) Create(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")

	var parsed service.ParsedSkill
	var files json.RawMessage

	switch {
	case strings.HasPrefix(ct, "multipart/form-data"):
		skill, fileMap, err := h.parseZipUpload(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		parsed = skill
		files = fileMap

	default:
		// JSON body
		var req createSkillRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Content == "" {
			writeError(w, http.StatusBadRequest, "content is required (raw SKILL.md text)")
			return
		}

		var err error
		parsed, err = service.ParseSkillMD(req.Content)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := service.ValidateFrontmatter(parsed.Frontmatter); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		files = json.RawMessage("{}")
	}

	wsID := workspaceIDFromCtx(r)

	// Check for name conflict within workspace
	existing, _ := h.repo.GetByName(r.Context(), wsID, parsed.Frontmatter.Name)
	if existing != nil {
		writeError(w, http.StatusConflict, "a skill with name '"+parsed.Frontmatter.Name+"' already exists")
		return
	}

	now := time.Now().UTC()
	skill := &domain.Skill{
		ID:            domain.NewSkillID(),
		WorkspaceID:   wsID,
		Name:          parsed.Frontmatter.Name,
		Description:   parsed.Frontmatter.Description,
		License:       parsed.Frontmatter.License,
		Compatibility: parsed.Frontmatter.Compatibility,
		AllowedTools:  parsed.Frontmatter.AllowedTools,
		Metadata:      parsed.Frontmatter.Metadata,
		Content:       parsed.Body,
		Files:         files,
		CreatedAt:     now,
		UpdatedAt:     now,
		Type:          "skill",
	}
	if skill.Metadata == nil {
		skill.Metadata = map[string]string{}
	}

	if err := h.repo.Create(r.Context(), skill); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create skill")
		return
	}

	writeJSON(w, http.StatusOK, skill)
}

func (h *SkillHandler) List(w http.ResponseWriter, r *http.Request) {
	lp := parseListParams(r)
	lp.WorkspaceID = workspaceIDFromCtx(r)
	params := domain.SkillListParams{
		ListParams: lp,
	}

	skills, nextPage, err := h.repo.List(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list skills")
		return
	}
	if skills == nil {
		skills = []*domain.Skill{}
	}

	writeJSON(w, http.StatusOK, domain.ListResponse[*domain.Skill]{
		Data:     skills,
		NextPage: nextPage,
	})
}

func (h *SkillHandler) Retrieve(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "skill_id")
	skill, err := h.repo.GetByID(r.Context(), id)
	if err != nil || skill.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "skill not found")
		return
	}
	writeJSON(w, http.StatusOK, skill)
}

func (h *SkillHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "skill_id")
	skill, err := h.repo.GetByID(r.Context(), id)
	if err != nil || skill.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "skill not found")
		return
	}

	ct := r.Header.Get("Content-Type")

	var parsed service.ParsedSkill
	var files json.RawMessage

	switch {
	case strings.HasPrefix(ct, "multipart/form-data"):
		s, f, err := h.parseZipUpload(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		parsed = s
		files = f

	default:
		var req createSkillRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Content == "" {
			writeError(w, http.StatusBadRequest, "content is required (raw SKILL.md text)")
			return
		}

		var err error
		parsed, err = service.ParseSkillMD(req.Content)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := service.ValidateFrontmatter(parsed.Frontmatter); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		files = skill.Files // keep existing files if JSON update
	}

	// If name changed, check for conflict within workspace
	if parsed.Frontmatter.Name != skill.Name {
		existing, _ := h.repo.GetByName(r.Context(), skill.WorkspaceID, parsed.Frontmatter.Name)
		if existing != nil {
			writeError(w, http.StatusConflict, "a skill with name '"+parsed.Frontmatter.Name+"' already exists")
			return
		}
	}

	skill.Name = parsed.Frontmatter.Name
	skill.Description = parsed.Frontmatter.Description
	skill.License = parsed.Frontmatter.License
	skill.Compatibility = parsed.Frontmatter.Compatibility
	skill.AllowedTools = parsed.Frontmatter.AllowedTools
	skill.Content = parsed.Body
	skill.Files = files
	if parsed.Frontmatter.Metadata != nil {
		skill.Metadata = parsed.Frontmatter.Metadata
	}

	if err := h.repo.Update(r.Context(), skill); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update skill")
		return
	}

	writeJSON(w, http.StatusOK, skill)
}

func (h *SkillHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "skill_id")
	skill, err := h.repo.GetByID(r.Context(), id)
	if err != nil || skill.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "skill not found")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete skill")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":      id,
		"deleted": true,
	})
}

const maxUploadSize = 10 * 1024 * 1024 // 10 MB

func (h *SkillHandler) parseZipUpload(r *http.Request) (service.ParsedSkill, json.RawMessage, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		return service.ParsedSkill{}, nil, err
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		return service.ParsedSkill{}, nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return service.ParsedSkill{}, nil, err
	}

	result, err := service.ParseSkillZip(data)
	if err != nil {
		return service.ParsedSkill{}, nil, err
	}

	filesJSON, err := service.FilesToJSON(result.Files)
	if err != nil {
		return service.ParsedSkill{}, nil, err
	}

	return result.Skill, filesJSON, nil
}

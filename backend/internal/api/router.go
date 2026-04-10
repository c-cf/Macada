package api

import (
	"net/http"

	"github.com/c-cf/macada/internal/api/handler"
	authmw "github.com/c-cf/macada/internal/api/middleware"
	"github.com/c-cf/macada/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Deps struct {
	EnvironmentHandler *handler.EnvironmentHandler
	AgentHandler       *handler.AgentHandler
	SessionHandler     *handler.SessionHandler
	EventHandler       *handler.EventHandler
	AnalyticsHandler   *handler.AnalyticsHandler
	SkillHandler       *handler.SkillHandler
	InternalHandler    *handler.InternalHandler
	LLMProxyHandler    *handler.LLMProxyHandler
	WorkspaceHandler   *handler.WorkspaceHandler
	APIKeyHandler      *handler.APIKeyHandler
	BootstrapHandler   *handler.BootstrapHandler
	AuthHandler        *handler.AuthHandler
	FileHandler        *handler.FileHandler
	ResourceHandler    *handler.ResourceHandler
	APIKeyRepo         domain.APIKeyRepository
	JWTValidator       authmw.TokenValidator
	MemberRepo         authmw.MembershipChecker
}

func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(corsMiddleware)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Public endpoints (no auth)
	r.Post("/v1/bootstrap", deps.BootstrapHandler.Bootstrap)
	r.Post("/v1/auth/register", deps.AuthHandler.Register)
	r.Post("/v1/auth/login", deps.AuthHandler.Login)

	// Authenticated API routes (JWT Bearer OR X-Api-Key)
	r.Route("/v1", func(r chi.Router) {
		r.Use(authmw.Auth(deps.APIKeyRepo, deps.JWTValidator, deps.MemberRepo))

		// Auth (authenticated)
		r.Get("/auth/me", deps.AuthHandler.Me)
		r.Get("/auth/workspaces", deps.AuthHandler.ListWorkspaces)

		// Workspaces
		r.Route("/workspaces", func(r chi.Router) {
			r.Get("/", deps.WorkspaceHandler.List)
			r.Route("/{workspace_id}", func(r chi.Router) {
				r.Get("/", deps.WorkspaceHandler.Retrieve)
				r.Post("/", deps.WorkspaceHandler.Update)
			})
		})

		// API Keys
		r.Route("/api-keys", func(r chi.Router) {
			r.Post("/", deps.APIKeyHandler.Create)
			r.Get("/", deps.APIKeyHandler.List)
			r.Route("/{key_id}", func(r chi.Router) {
				r.Post("/revoke", deps.APIKeyHandler.Revoke)
				r.Delete("/", deps.APIKeyHandler.Delete)
			})
		})

		// Environments
		r.Route("/environments", func(r chi.Router) {
			r.Post("/", deps.EnvironmentHandler.Create)
			r.Get("/", deps.EnvironmentHandler.List)
			r.Route("/{environment_id}", func(r chi.Router) {
				r.Get("/", deps.EnvironmentHandler.Retrieve)
				r.Post("/", deps.EnvironmentHandler.Update)
				r.Delete("/", deps.EnvironmentHandler.Delete)
				r.Post("/archive", deps.EnvironmentHandler.Archive)
			})
		})

		// Skills
		r.Route("/skills", func(r chi.Router) {
			r.Post("/", deps.SkillHandler.Create)
			r.Get("/", deps.SkillHandler.List)
			r.Route("/{skill_id}", func(r chi.Router) {
				r.Get("/", deps.SkillHandler.Retrieve)
				r.Post("/", deps.SkillHandler.Update)
				r.Delete("/", deps.SkillHandler.Delete)
			})
		})

		// Files
		r.Route("/files", func(r chi.Router) {
			r.Post("/", deps.FileHandler.Upload)
			r.Get("/", deps.FileHandler.List)
			r.Route("/{file_id}", func(r chi.Router) {
				r.Get("/", deps.FileHandler.GetMetadata)
				r.Get("/content", deps.FileHandler.Download)
				r.Delete("/", deps.FileHandler.Delete)
			})
		})

		// Agents
		r.Route("/agents", func(r chi.Router) {
			r.Post("/", deps.AgentHandler.Create)
			r.Get("/", deps.AgentHandler.List)
			r.Route("/{agent_id}", func(r chi.Router) {
				r.Get("/", deps.AgentHandler.Retrieve)
				r.Post("/", deps.AgentHandler.Update)
				r.Post("/archive", deps.AgentHandler.Archive)
			})
		})

		// Analytics
		r.Route("/analytics", func(r chi.Router) {
			r.Get("/usage", deps.AnalyticsHandler.Usage)
			r.Get("/logs", deps.AnalyticsHandler.Logs)
		})

		// Sessions
		r.Route("/sessions", func(r chi.Router) {
			r.Post("/", deps.SessionHandler.Create)
			r.Get("/", deps.SessionHandler.List)
			r.Route("/{session_id}", func(r chi.Router) {
				r.Get("/", deps.SessionHandler.Retrieve)
				r.Post("/archive", deps.SessionHandler.Archive)

				// Resources
				r.Route("/resources", func(r chi.Router) {
					r.Post("/", deps.ResourceHandler.Add)
					r.Get("/", deps.ResourceHandler.List)
					r.Route("/{resource_id}", func(r chi.Router) {
						r.Get("/", deps.ResourceHandler.Retrieve)
						r.Post("/", deps.ResourceHandler.Update)
						r.Delete("/", deps.ResourceHandler.Delete)
					})
				})

				// Events
				r.Route("/events", func(r chi.Router) {
					r.Post("/", deps.EventHandler.Send)
					r.Get("/", deps.EventHandler.List)
					r.Get("/stream", deps.EventHandler.Stream)
				})
			})
		})
	})

	// Internal routes (sandbox runtime → control plane)
	// InternalOnly: reject requests from non-private IPs (only Docker network / localhost allowed)
	r.Route("/internal/v1/sandbox/{session_id}", func(r chi.Router) {
		r.Use(authmw.InternalOnly)
		r.Post("/events", deps.InternalHandler.IngestEvents)
		r.Post("/files", deps.InternalHandler.UploadFile)
		r.Post("/llm", deps.LLMProxyHandler.ProxyLLM)
	})

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Api-Key, X-Admin-Secret, X-Workspace-Id, anthropic-version, anthropic-beta")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

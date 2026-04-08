package middleware

import "context"

type ctxKey string

const (
	ctxKeyWorkspaceID ctxKey = "workspace_id"
	ctxKeyAPIKeyID    ctxKey = "api_key_id"
	ctxKeyUserID      ctxKey = "user_id"
)

func WorkspaceIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyWorkspaceID).(string)
	return v
}

func APIKeyIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyAPIKeyID).(string)
	return v
}

func WithWorkspaceID(ctx context.Context, wsID string) context.Context {
	return context.WithValue(ctx, ctxKeyWorkspaceID, wsID)
}

func WithAPIKeyID(ctx context.Context, keyID string) context.Context {
	return context.WithValue(ctx, ctxKeyAPIKeyID, keyID)
}

func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyUserID).(string)
	return v
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, userID)
}

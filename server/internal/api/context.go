package api

import "context"

type contextKey string

const (
	ctxKeyPlayerID contextKey = "player_id"
)

// WithPlayerID stores a player ID in the context.
func WithPlayerID(ctx context.Context, playerID string) context.Context {
	return context.WithValue(ctx, ctxKeyPlayerID, playerID)
}

// GetPlayerID extracts the player ID from the context.
func GetPlayerID(ctx context.Context) string {
	if id, ok := ctx.Value(ctxKeyPlayerID).(string); ok {
		return id
	}
	return "anonymous"
}

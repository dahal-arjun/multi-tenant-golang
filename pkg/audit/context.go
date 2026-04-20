package audit

import "context"

type ctxKey struct{}

var actorIDKey ctxKey

// WithActorID attaches the authenticated user's database ID (users.id) for GORM callbacks / services.
func WithActorID(ctx context.Context, userID uint) context.Context {
	return context.WithValue(ctx, actorIDKey, userID)
}

// ActorID returns the actor user ID if present.
func ActorID(ctx context.Context) (uint, bool) {
	v, ok := ctx.Value(actorIDKey).(uint)
	return v, ok
}

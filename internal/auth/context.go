package auth

import "context"

type sessionKey struct{}

func WithSession(ctx context.Context, sub, email string) context.Context {
	return context.WithValue(ctx, sessionKey{}, sessionPayload{Sub: sub, Email: email})
}

func Subject(ctx context.Context) string {
	p, _ := ctx.Value(sessionKey{}).(sessionPayload)
	return p.Sub
}

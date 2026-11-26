package auth

import (
	"context"
	"strings"
)

type sessionKey struct{}

func WithSession(ctx context.Context, sub, email string) context.Context {
	return context.WithValue(ctx, sessionKey{}, sessionPayload{Sub: sub, Email: email})
}

func Subject(ctx context.Context) string {
	p, _ := ctx.Value(sessionKey{}).(sessionPayload)
	return p.Sub
}

// CreatedBy prefers the session email, then the subject, then a wall identity.
func CreatedBy(ctx context.Context) string {
	p, _ := ctx.Value(sessionKey{}).(sessionPayload)
	if email := strings.TrimSpace(p.Email); email != "" {
		return email
	}
	if sub := strings.TrimSpace(p.Sub); sub != "" {
		return sub
	}
	return "alert-wall"
}

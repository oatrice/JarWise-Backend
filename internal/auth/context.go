package auth

import (
	"context"
	"jarwise-backend/internal/models"
)

type contextKey string

const userContextKey contextKey = "authenticated-user"

func ContextWithUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(userContextKey).(*models.User)
	return user, ok && user != nil
}

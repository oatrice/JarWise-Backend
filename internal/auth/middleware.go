package auth

import (
	"errors"
	"net/http"
)

func RequireAuth(service *Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}

		user, err := service.AuthenticateRequest(r.Context(), cookie.Value)
		if err != nil {
			if errors.Is(err, ErrUnauthorized) {
				service.ClearSessionCookie(w)
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
			http.Error(w, "failed to authenticate request", http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r.WithContext(ContextWithUser(r.Context(), user)))
	})
}

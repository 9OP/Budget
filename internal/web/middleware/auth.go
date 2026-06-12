package middleware

import (
	"net/http"

	"github.com/9op/budget/internal/auth"
)

// RequireAuth validates the session JWT cookie and injects user_id into the request context.
// Unauthenticated or expired requests are redirected to /login.
func RequireAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("budget_session")
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			userID, err := auth.ValidateToken(cookie.Value, jwtSecret)
			if err != nil {
				http.SetCookie(w, &http.Cookie{Name: "budget_session", MaxAge: -1, Path: "/"})
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithUserID(r.Context(), userID)))
		})
	}
}

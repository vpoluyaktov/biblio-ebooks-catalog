package auth

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"opds-server/internal/db"
)

type contextKey string

const UserContextKey contextKey = "user"

func GetUserFromContext(ctx context.Context) *db.User {
	user, ok := ctx.Value(UserContextKey).(*db.User)
	if !ok {
		return nil
	}
	return user
}

func (a *Auth) SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		user, err := a.ValidateSession(cookie.Value)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Auth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r.Context())
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *Auth) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r.Context())
		if user == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Not authenticated. Please log in again."}`))
			return
		}
		if !user.IsAdmin() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"Admin access required"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *Auth) BasicAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if already authenticated via session
		if GetUserFromContext(r.Context()) != nil {
			next.ServeHTTP(w, r)
			return
		}

		// Try Basic Auth
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			a.requestBasicAuth(w)
			return
		}

		if !strings.HasPrefix(authHeader, "Basic ") {
			a.requestBasicAuth(w)
			return
		}

		decoded, err := base64.StdEncoding.DecodeString(authHeader[6:])
		if err != nil {
			a.requestBasicAuth(w)
			return
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			a.requestBasicAuth(w)
			return
		}

		user, err := a.Authenticate(parts[0], parts[1])
		if err != nil {
			a.requestBasicAuth(w)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Auth) OptionalBasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if already authenticated via session
		if GetUserFromContext(r.Context()) != nil {
			next.ServeHTTP(w, r)
			return
		}

		// Try Basic Auth if provided
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Basic ") {
			decoded, err := base64.StdEncoding.DecodeString(authHeader[6:])
			if err == nil {
				parts := strings.SplitN(string(decoded), ":", 2)
				if len(parts) == 2 {
					user, err := a.Authenticate(parts[0], parts[1])
					if err == nil {
						ctx := context.WithValue(r.Context(), UserContextKey, user)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (a *Auth) requestBasicAuth(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="opds-server"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

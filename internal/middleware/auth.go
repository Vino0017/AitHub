package middleware

import (
	"github.com/skillhub/api/internal/config"
	"context"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/crypto"
	"github.com/skillhub/api/internal/helpers"
)

type contextKey string

const (
	CtxTokenID       contextKey = "token_id"
	CtxNamespaceID   contextKey = "namespace_id"
	CtxNamespaceName contextKey = "namespace_name"
	CtxNamespaceType contextKey = "namespace_type"
	CtxIsAnonymous   contextKey = "is_anonymous"
)

// Auth validates the Bearer token and injects token/namespace info into context.
func Auth(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				helpers.WriteError(w, http.StatusUnauthorized, "missing_token", "Authorization header with Bearer token required", "")
				return
			}

			hash := crypto.HashToken(token)

			var tokenID, nsID interface{}
			var nsName, nsType *string
			var nsBanned *bool

			err := pool.QueryRow(r.Context(),
				`SELECT t.id, t.namespace_id, n.name, n.type, n.banned
				 FROM tokens t
				 LEFT JOIN namespaces n ON t.namespace_id = n.id
				 WHERE t.token_hash = $1`, hash).Scan(&tokenID, &nsID, &nsName, &nsType, &nsBanned)

			if err != nil {
				helpers.WriteError(w, http.StatusUnauthorized, "invalid_token", "Token not found or expired", "")
				return
			}

			if nsBanned != nil && *nsBanned {
				helpers.WriteError(w, http.StatusForbidden, "namespace_banned", "This namespace has been banned", "")
				return
			}

			// Increment daily uses
			go pool.Exec(context.Background(),
				`UPDATE tokens SET daily_uses = daily_uses + 1, last_used = NOW() WHERE id = $1`, tokenID)

			ctx := context.WithValue(r.Context(), CtxTokenID, tokenID)
			isAnonymous := nsID == nil
			ctx = context.WithValue(ctx, CtxIsAnonymous, isAnonymous)

			if !isAnonymous {
				ctx = context.WithValue(ctx, CtxNamespaceID, nsID)
				if nsName != nil {
					ctx = context.WithValue(ctx, CtxNamespaceName, *nsName)
				}
				if nsType != nil {
					ctx = context.WithValue(ctx, CtxNamespaceType, *nsType)
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireNamespace rejects requests from anonymous tokens.
func RequireNamespace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAnon, _ := r.Context().Value(CtxIsAnonymous).(bool)
		if isAnon {
			helpers.WriteError(w, http.StatusForbidden, "namespace_required",
				"A registered namespace is required for this action",
				"Ask your human to run: bash <(curl -fsSL " + config.GetDomain() + "/install) --register --github")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// AdminAuth validates the admin token from environment variable.
func AdminAuth(adminToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token != adminToken {
				helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid admin token", "")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}



// GetTokenID gets the token ID from context.
func GetTokenID(ctx context.Context) interface{} {
	return ctx.Value(CtxTokenID)
}

// GetNamespaceID gets the namespace ID from context.
func GetNamespaceID(ctx context.Context) interface{} {
	return ctx.Value(CtxNamespaceID)
}

// GetNamespaceName gets the namespace name from context.
func GetNamespaceName(ctx context.Context) string {
	v, _ := ctx.Value(CtxNamespaceName).(string)
	return v
}

// IsAnonymous returns true if the current request is from an anonymous token.
func IsAnonymous(ctx context.Context) bool {
	v, _ := ctx.Value(CtxIsAnonymous).(bool)
	return v
}

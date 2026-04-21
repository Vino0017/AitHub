package middleware

import (
	"github.com/skillhub/api/internal/config"
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
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

// OptionalAuth validates the Bearer token if present, but allows anonymous access.
func OptionalAuth(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				// No token provided - allow anonymous access
				ctx := context.WithValue(r.Context(), CtxIsAnonymous, true)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			hash := crypto.HashToken(token)

			var tokenIDRaw, nsIDRaw interface{}
			var nsName, nsType *string
			var nsBanned *bool

			err := pool.QueryRow(r.Context(),
				`SELECT t.id, t.namespace_id, n.name, n.type, n.banned
				 FROM tokens t
				 LEFT JOIN namespaces n ON t.namespace_id = n.id
				 WHERE t.token_hash = $1`, hash).Scan(&tokenIDRaw, &nsIDRaw, &nsName, &nsType, &nsBanned)

			if err != nil {
				// Invalid token - still allow anonymous access
				ctx := context.WithValue(r.Context(), CtxIsAnonymous, true)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if nsBanned != nil && *nsBanned {
				helpers.WriteError(w, http.StatusForbidden, "namespace_banned", "This namespace has been banned", "")
				return
			}

			tokenID := parseUUID(tokenIDRaw)

			// Increment daily uses
			go pool.Exec(context.Background(),
				`UPDATE tokens SET daily_uses = daily_uses + 1, last_used = NOW() WHERE id = $1`, tokenID)

			ctx := context.WithValue(r.Context(), CtxTokenID, tokenID)
			isAnonymous := nsIDRaw == nil
			ctx = context.WithValue(ctx, CtxIsAnonymous, isAnonymous)

			if !isAnonymous {
				ctx = context.WithValue(ctx, CtxNamespaceID, parseUUID(nsIDRaw))
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

// Auth validates the Bearer token and injects token/namespace info into context.
// Requires a valid token - rejects anonymous access.
func Auth(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				helpers.WriteError(w, http.StatusUnauthorized, "missing_token", "Authorization header with Bearer token required", "")
				return
			}

			hash := crypto.HashToken(token)

			var tokenIDRaw, nsIDRaw interface{}
			var nsName, nsType *string
			var nsBanned *bool

			err := pool.QueryRow(r.Context(),
				`SELECT t.id, t.namespace_id, n.name, n.type, n.banned
				 FROM tokens t
				 LEFT JOIN namespaces n ON t.namespace_id = n.id
				 WHERE t.token_hash = $1`, hash).Scan(&tokenIDRaw, &nsIDRaw, &nsName, &nsType, &nsBanned)

			if err != nil {
				helpers.WriteError(w, http.StatusUnauthorized, "invalid_token", "Token not found or expired", "")
				return
			}

			if nsBanned != nil && *nsBanned {
				helpers.WriteError(w, http.StatusForbidden, "namespace_banned", "This namespace has been banned", "")
				return
			}

			tokenID := parseUUID(tokenIDRaw)

			// Increment daily uses
			go pool.Exec(context.Background(),
				`UPDATE tokens SET daily_uses = daily_uses + 1, last_used = NOW() WHERE id = $1`, tokenID)

			ctx := context.WithValue(r.Context(), CtxTokenID, tokenID)
			isAnonymous := nsIDRaw == nil
			ctx = context.WithValue(ctx, CtxIsAnonymous, isAnonymous)

			if !isAnonymous {
				ctx = context.WithValue(ctx, CtxNamespaceID, parseUUID(nsIDRaw))
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

// parseUUID converts pgx scan result (interface{}) to uuid.UUID.
// pgx v5 scans PostgreSQL UUID as [16]byte when target is interface{}.
func parseUUID(v interface{}) uuid.UUID {
	if v == nil {
		return uuid.UUID{}
	}
	switch val := v.(type) {
	case [16]byte:
		return uuid.UUID(val)
	case uuid.UUID:
		return val
	case string:
		parsed, _ := uuid.Parse(val)
		return parsed
	case []byte:
		if len(val) == 16 {
			var u uuid.UUID
			copy(u[:], val)
			return u
		}
		parsed, _ := uuid.ParseBytes(val)
		return parsed
	default:
		return uuid.UUID{}
	}
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

package middlewares

import (
	"Sellora-Backend/internal/httpx"
	"Sellora-Backend/internal/models"
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type claimsContextKey string

const claimsKey claimsContextKey = "auth_claims"

// RequireAuth validates a Bearer JWT and stores models.Claims in context.
// Usage: r.With(middlewares.RequireAuth).Post("/products", handler)
func RequireAuth(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		header := strings.TrimSpace(
			r.Header.Get("Authorization"),
		)

		if header == "" {
			httpx.SendJSON(
				w, http.StatusUnauthorized,
				map[string]any{
					"error": "missing authorization header",
				},
			)
			return
		}

		parts := strings.SplitN(header, " ", 2)

		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			httpx.SendJSON(
				w, http.StatusUnauthorized,
				map[string]any{
					"error": "invalid authorization header",
				},
			)
			return
		}

		tokenString := strings.TrimSpace(parts[1])

		if tokenString == "" {
			httpx.SendJSON(
				w, http.StatusUnauthorized,
				map[string]any{
					"error": "missing token",
				},
			)
			return
		}

		secret := os.Getenv("JWT_SECRET")

		if len(secret) == 0 {
			httpx.SendJSON(
				w, http.StatusInternalServerError,
				map[string]any{
					"error": "server misconfigured",
				},
			)
			return
		}

		claims := &models.Claims{}

		token, err := jwt.ParseWithClaims(
			tokenString,
			claims,
			func(t *jwt.Token) (any, error) {
				if t.Method != jwt.SigningMethodHS256 {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			},
		)

		if err != nil || !token.Valid {
			msg := "invalid token"

			if err != nil &&
				(strings.Contains(err.Error(), "expired") ||
					strings.Contains(err.Error(), "exp")) {
				msg = "token expired"
			}

			httpx.SendJSON(
				w, http.StatusUnauthorized,
				map[string]any{
					"error": msg,
				},
			)
			return
		}

		if claims.UserID == 0 {
			httpx.SendJSON(
				w, http.StatusUnauthorized,
				map[string]any{
					"error": "invalid token",
				},
			)
			return
		}

		ctx := context.WithValue(
			r.Context(),
			claimsKey,
			claims,
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ClaimsFromContext returns the authenticated models.Claims.
func ClaimsFromContext(
	ctx context.Context,
) (*models.Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*models.Claims)

	if !ok || claims == nil {
		return nil, false
	}

	return claims, true
}

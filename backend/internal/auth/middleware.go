package auth

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
	"github.com/orgmelethil/playhub/backend/internal/middleware"
)

// Authenticator is the behaviour the middleware needs from the service.
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (User, error)
}

// RequireAuth rejects any request without a valid access token and puts the
// authenticated user on the request context for the handlers below it.
//
// It is a middleware rather than a call inside each handler so that a new
// protected route cannot be added without the check: the route is mounted
// behind the wrapper or it is not protected at all.
func RequireAuth(authenticator Authenticator) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := bearerToken(r)
			if err != nil {
				unauthorised(w, r, "A bearer access token is required.")
				return
			}

			user, err := authenticator.Authenticate(r.Context(), token)
			switch {
			case err == nil:
			case errors.Is(err, ErrInvalidToken):
				unauthorised(w, r, "The access token is invalid or has expired.")
				return
			case errors.Is(err, ErrAccountInactive):
				httpx.Error(w, r, http.StatusForbidden, "account_inactive", "This account has been deactivated.")
				return
			default:
				httpx.Error(w, r, http.StatusInternalServerError, "internal_error", "Something went wrong.")
				return
			}

			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), user)))
		})
	}
}

// RequireRole rejects an authenticated request whose user holds none of the
// allowed roles. It must be mounted inside RequireAuth; a request that reaches
// it without a principal is a wiring mistake and is refused rather than let
// through.
func RequireRole(allowed ...Role) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := PrincipalFromContext(r.Context())
			if !ok {
				unauthorised(w, r, "A bearer access token is required.")
				return
			}

			if !slices.Contains(allowed, user.Role) {
				httpx.Error(w, r, http.StatusForbidden, "forbidden",
					"Your account does not have access to this resource.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// bearerToken pulls the credential out of the Authorization header.
func bearerToken(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", ErrInvalidToken
	}

	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, bearerScheme) {
		return "", ErrInvalidToken
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", ErrInvalidToken
	}
	return token, nil
}

// unauthorised writes a 401 with the WWW-Authenticate header the status code
// requires.
func unauthorised(w http.ResponseWriter, r *http.Request, message string) {
	w.Header().Set("WWW-Authenticate", bearerScheme)
	httpx.Error(w, r, http.StatusUnauthorized, "unauthorized", message)
}

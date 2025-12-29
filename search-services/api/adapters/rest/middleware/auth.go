package middleware

import (
	"net/http"
	"strings"
)

type TokenVerifier interface {
	Verify(token string) error
	RefreshAccessToken(refreshToken string) (string, error)
}

func Auth(next http.HandlerFunc, verifier TokenVerifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(r.Header.Get("Authorization"))
		
		var accessToken string
		if len(parts) == 2 && (parts[0] == "Bearer" || parts[0] == "Token") {
			accessToken = parts[1]
		}

		if accessToken == "" || verifier.Verify(accessToken) != nil {
			cookie, err := r.Cookie("refresh_token")
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			newAccessToken, err := verifier.RefreshAccessToken(cookie.Value)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			r.Header.Set("Authorization", "Bearer "+newAccessToken)
			accessToken = newAccessToken
		}

		if err := verifier.Verify(accessToken); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}

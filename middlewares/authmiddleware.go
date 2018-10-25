package middlewares

import (
	"log"
	"net/http"
)

type AuthMiddleware struct {
	tokenUsers map[string]string
}

func NewAuthMiddleware() *AuthMiddleware {
	tokenUsers := make(map[string]string)
	tokenUsers["00000000"] = "user0"
	tokenUsers["aaaaaaaa"] = "userA"
	tokenUsers["05f717e5"] = "randomUser"
	tokenUsers["deadbeef"] = "user0"

	return &AuthMiddleware{
		tokenUsers: tokenUsers,
	}
}

func (awm *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Session-Token")

		if user, found := awm.tokenUsers[token]; found {
			log.Printf("Found User: %v\n", user)
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "UnAuthorized Error", http.StatusForbidden)
		}
	})
}

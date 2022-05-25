package authorization

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt"
	"net/http"
	"strings"
	"wb_Bar/pkg/models"
)

func Jwt() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bearer := r.Header.Get("Authorization")
			s := strings.Split(bearer, " ")

			if len(s) != 2 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			token, errParseToken := jwt.ParseWithClaims(s[1], &models.UserWithClaims{}, func(t *jwt.Token) (interface{}, error) {
				return []byte(models.SigningKey), nil
			})
			if errParseToken != nil {
				fmt.Println("parse token failed: ", errParseToken)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if !token.Valid {
				fmt.Println("!valid")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(*models.UserWithClaims)
			if !ok {
				fmt.Println("claims: ", ok)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), models.CtxKey(), *claims.UserAuthData()))
			next.ServeHTTP(w, r)
		})
	}
}

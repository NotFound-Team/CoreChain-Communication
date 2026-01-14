package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"corechain-communication/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

func WithAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("incoming request", r.URL.Path)
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized: No token provided", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Unauthorized: Invalid token format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]
		jwtSecret := config.Get().JwtSecret

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Unauthorized: Invalid token claims", http.StatusUnauthorized)
			return
		}

		userID, _ := claims["_id"].(string)

		userName, _ := claims["name"].(string)

		roleName := ""
		if roleMap, ok := claims["role"].(map[string]interface{}); ok {
			if rName, ok := roleMap["name"].(string); ok {
				roleName = rName
			}
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "user_id", userID)
		ctx = context.WithValue(ctx, "user_name", userName)
		ctx = context.WithValue(ctx, "user_role", roleName)
		log.Println("user_id", userID)
		log.Println("user_name", userName)
		log.Println("user_role", roleName)
		next(w, r.WithContext(ctx))
	}
}

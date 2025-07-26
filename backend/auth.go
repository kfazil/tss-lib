package main

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/genv"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("nokey_secret")

func GenerateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func AuthMiddleware(r *ghttp.Request) {
	tokenStr := r.Header.Get("Authorization")
	// If IGNORE_CODESPACE_AUTH is set, ignore Authorization header if it looks like a Codespace token
	if genv.Get("IGNORE_CODESPACE_AUTH", "false").String() == "true" {
		if tokenStr != "" && (tokenStr == "Bearer github" || tokenStr == "Bearer codespace" || len(tokenStr) < 32) {
			tokenStr = ""
		}
	}
	if tokenStr == "" {
		r.Response.WriteJson(g.Map{"error": "Missing token"})
		r.Exit()
		return
	}
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		r.Response.WriteJson(g.Map{"error": "Invalid token"})
		r.Exit()
		return
	}
	claims := token.Claims.(jwt.MapClaims)
	r.SetCtxVar("user_id", claims["user_id"])
}

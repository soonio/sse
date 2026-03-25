package middleware

import (
	"context"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"pusher/internal/core/limiter"
)

// JWTClaims JWT 载荷。
type JWTClaims struct {
	UserID int64 `json:"uid"`
	jwtClaims
}

type jwtClaims struct {
	Exp int64 `json:"exp"`
	Iat int64 `json:"iat"`
}

// Auth 认证 + 频率限制中间件。
func Auth(logger *zap.Logger, lim *limiter.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bt := r.Header.Get("Authorization")

			if !strings.HasPrefix(bt, "Bearer ") {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(bt, "Bearer ")

			if lim.Has(token) {
				w.WriteHeader(http.StatusNotAcceptable)
				_, _ = w.Write([]byte("frequency limit"))
				return
			}
			_ = lim.Have(token, 3)

			claims, err := parseJWT(token)
			if err != nil {
				logger.Debug("JWT 解析失败", zap.Error(err))
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("invalid token"))
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "UID", claims.UserID)))
		})
	}
}

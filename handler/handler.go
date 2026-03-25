package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"

	"pusher/hub"
	"pusher/limiter"
	"pusher/pool"
	"pusher/types"
)

type JWTClaims struct {
	UserID int64 `json:"uid"`
	jwtClaims
}

type jwtClaims struct {
	Exp int64 `json:"exp"`
	Iat int64 `json:"iat"`
}

type JWTConfig struct {
	Secret string
}

func parseJWT(tokenString string) (*JWTClaims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	payload, err := base64RawURLEncode(parts[1])
	if err != nil {
		return nil, err
	}

	var claims JWTClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}

	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return nil, errors.New("token expired")
	}

	return &claims, nil
}

func base64RawURLEncode(data string) ([]byte, error) {
	if l := len(data) % 4; l > 0 {
		data += strings.Repeat("=", 4-l)
	}
	return base64.URLEncoding.DecodeString(data)
}

func NewHandler(debug bool, logger *zap.Logger, h *hub.Hub, lim *limiter.Limiter, mp pool.Pool[*types.Message[string]]) *chi.Mux {

	var authorization = func(next http.Handler) http.Handler {
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

	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(authorization)
		r.Get("/listen", Listen(logger, h, mp))
	})
	r.Mount("/eqw2025ddffxce", middleware.Profiler())
	if debug {
		r.Post("/online", Online(h))
		r.Post("/send/{uid}", Send(h))
		r.Post("/close", Close())
	}
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "404 迷路了吗", http.StatusNotFound)
	})

	return r
}

func Close() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}

func Listen(logger *zap.Logger, h *hub.Hub, mp pool.Pool[*types.Message[string]]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()

		uid := strconv.FormatInt(r.Context().Value("UID").(int64), 10)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		quit := make(chan struct{})
		timestamp := time.Now().UnixNano()
		h.Store(uid, timestamp, w, quit)
		defer h.Remove(uid, timestamp)

		timeout := time.NewTimer(30 * time.Minute)
		defer timeout.Stop()

		go func() {
			if c, ok := h.Get(uid); ok {
				msg := mp.Get()
				defer mp.Put(msg)
				msg.Body = uid
				msg.Timestamp = time.Now().Unix()
				if err := c.Push(msg); err != nil {
					logger.Error(err.Error())
				}
			}
		}()

		select {
		case <-quit:
			return
		case <-timeout.C:
			return
		case <-r.Context().Done():
			return
		}
	}
}

func Send(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid := chi.URLParam(r, "uid")

		msg, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		if uid == "" {
			w.WriteHeader(422)
			_, _ = w.Write([]byte("参数错误"))
		} else {
			if c, ok := h.Get(uid); ok {
				_ = c.Push(bytes.NewBuffer(msg))
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("user offline"))
			}
		}
	}
}

func Online(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bs, _ := jsoniter.Marshal(h.Keys())

		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bs)
	}
}

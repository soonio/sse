package middleware

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

func parseJWT(tokenString string) (*JWTClaims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	payload, err := base64RawURLDecode(parts[1])
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

func base64RawURLDecode(data string) ([]byte, error) {
	if l := len(data) % 4; l > 0 {
		data += strings.Repeat("=", 4-l)
	}
	return base64.URLEncoding.DecodeString(data)
}

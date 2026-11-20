package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const sessionCookieName = "alertwall_session"

type sessionPayload struct {
	Sub   string    `json:"sub"`
	Email string    `json:"email,omitempty"`
	Exp   time.Time `json:"exp"`
}

func signSession(secret []byte, p sessionPayload) (string, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(raw)
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(raw) + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func parseSession(secret []byte, value string) (sessionPayload, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return sessionPayload{}, errors.New("invalid session format")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return sessionPayload{}, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return sessionPayload{}, err
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(raw)
	if !hmac.Equal(mac.Sum(nil), sig) {
		return sessionPayload{}, errors.New("invalid session signature")
	}
	var p sessionPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return sessionPayload{}, err
	}
	if time.Now().After(p.Exp) {
		return sessionPayload{}, errors.New("session expired")
	}
	return p, nil
}
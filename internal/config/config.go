package config

import (
	"encoding/base64"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/discovery"
	"github.com/vutratenko/shturval-alert-wall/internal/mood"
)

type AuthConfig struct {
	Enabled        bool
	RedirectURL    string
	OIDCClientName string
	Namespace      string
	SessionTTL     time.Duration
	CookieSecret   []byte
}

type Config struct {
	HTTPBind         string
	PollInterval     time.Duration
	SourceMode       string
	VMSelectURL      string
	FakeAlertsPath   string
	StaticSources    []discovery.StaticEndpoint
	Mood             mood.Thresholds
	DiscoveryEvery   time.Duration
	PollTimeout      time.Duration
	DiscoveryEnabled bool
	Auth             AuthConfig
}

func Load() Config {
	cfg := Config{
		HTTPBind:         env("HTTP_BIND_ADDRESS", ":8080"),
		PollInterval:     durationEnv("POLL_INTERVAL", 10*time.Second),
		SourceMode:       strings.ToLower(env("SOURCE_MODE", "alertmanager")),
		VMSelectURL:      env("VMSELECT_URL", ""),
		FakeAlertsPath:   env("FAKE_ALERTS_PATH", ""),
		DiscoveryEvery:   durationEnv("DISCOVERY_INTERVAL", 60*time.Second),
		PollTimeout:      durationEnv("POLL_TIMEOUT", 5*time.Second),
		DiscoveryEnabled: env("DISCOVERY_ENABLED", "true") == "true",
		Mood:             mood.DefaultThresholds(),
		Auth: AuthConfig{
			Enabled:        env("AUTH_ENABLED", "false") == "true",
			RedirectURL:    env("AUTH_REDIRECT_URL", ""),
			OIDCClientName: env("AUTH_OIDC_CLIENT_NAME", "shturval-alert-wall-oidc"),
			Namespace:      env("POD_NAMESPACE", "shturval-alert-wall"),
			SessionTTL:     durationEnv("AUTH_SESSION_TTL", 24*time.Hour),
		},
	}
	if b64 := env("AUTH_COOKIE_SECRET", ""); b64 != "" {
		if b, err := base64.StdEncoding.DecodeString(b64); err == nil {
			cfg.Auth.CookieSecret = b
		}
	}
	if v := env("MOOD_MAX_WARNING_CALM", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Mood.MaxWarningForCalm = n
		}
	}
	if v := env("MOOD_MIN_CRITICAL_PANIC", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Mood.MinCriticalPanic = n
		}
	}
	if v := env("MOOD_MIN_WARNING_PANIC", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Mood.MinWarningPanic = n
		}
	}
	if raw := env("STATIC_SOURCES", ""); raw != "" {
		for _, part := range strings.Split(raw, ";") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				cfg.StaticSources = append(cfg.StaticSources, discovery.StaticEndpoint{Cluster: kv[0], URL: kv[1]})
			}
		}
	}
	return cfg
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func durationEnv(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
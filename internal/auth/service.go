package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type StaticConfig struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	SessionTTL   time.Duration
	CookieSecret []byte
	HTTPClient   *http.Client
}

// Service handles OIDC login and session cookies.
type Service struct {
	provider   *oidc.Provider
	verifier   *oidc.IDTokenVerifier
	oauth2     oauth2.Config
	secret     []byte
	ttl        time.Duration
	httpClient *http.Client
}

func NewService(ctx context.Context, cfg StaticConfig) (*Service, error) {
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = 24 * time.Hour
	}
	if len(cfg.CookieSecret) == 0 {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		cfg.CookieSecret = b
	}
	client := cfg.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	ctx = oidc.ClientContext(ctx, client)
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, err
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})
	return &Service{
		provider:   provider,
		verifier:   verifier,
		httpClient: client,
		oauth2: oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		secret: cfg.CookieSecret,
		ttl:    cfg.SessionTTL,
	}, nil
}

func (s *Service) Wrap(next http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /oauth2/login", s.handleLogin)
	mux.HandleFunc("GET /oauth2/callback", s.handleCallback)
	mux.HandleFunc("GET /oauth2/logout", s.handleLogout)
	mux.Handle("/", s.middleware(next))
	return mux
}

func (s *Service) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if _, err := s.sessionFromRequest(r); err != nil {
			if isAPIPath(r.URL.Path) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			rd := r.URL.Path
			if r.URL.RawQuery != "" {
				rd += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, "/oauth2/login?rd="+url.QueryEscape(rd), http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

const pkceCookie = "alertwall_pkce"

type pkcePayload struct {
	State    string `json:"state"`
	Verifier string `json:"verifier"`
	RD       string `json:"rd"`
}

func (s *Service) handleLogin(w http.ResponseWriter, r *http.Request) {
	verifier := oauth2.GenerateVerifier()
	state, _ := randToken()
	rd := r.URL.Query().Get("rd")
	if rd == "" {
		rd = "/wall"
	}
	payload, _ := json.Marshal(pkcePayload{State: state, Verifier: verifier, RD: rd})
	http.SetCookie(w, &http.Cookie{
		Name:     pkceCookie,
		Value:    base64.RawURLEncoding.EncodeToString(payload),
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil || isHTTPS(r),
	})
	url := s.oauth2.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	http.Redirect(w, r, url, http.StatusFound)
}

func (s *Service) handleCallback(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(pkceCookie)
	if err != nil {
		http.Error(w, "missing pkce cookie", http.StatusBadRequest)
		return
	}
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		http.Error(w, "bad pkce cookie", http.StatusBadRequest)
		return
	}
	var pkce pkcePayload
	if err := json.Unmarshal(raw, &pkce); err != nil {
		http.Error(w, "bad pkce payload", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != pkce.State {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}
	exCtx := r.Context()
	if s.httpClient != nil {
		exCtx = context.WithValue(exCtx, oauth2.HTTPClient, s.httpClient)
	}
	tok, err := s.oauth2.Exchange(exCtx, r.URL.Query().Get("code"), oauth2.VerifierOption(pkce.Verifier))
	if err != nil {
		http.Error(w, "token exchange failed", http.StatusUnauthorized)
		return
	}
	rawID, ok := tok.Extra("id_token").(string)
	if !ok || rawID == "" {
		http.Error(w, "missing id_token", http.StatusUnauthorized)
		return
	}
	idTok, err := s.verifier.Verify(r.Context(), rawID)
	if err != nil {
		http.Error(w, "invalid id_token", http.StatusUnauthorized)
		return
	}
	var claims struct {
		Email string `json:"email"`
	}
	_ = idTok.Claims(&claims)
	sess, err := signSession(s.secret, sessionPayload{
		Sub:   idTok.Subject,
		Email: claims.Email,
		Exp:   time.Now().Add(s.ttl),
	})
	if err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess,
		Path:     "/",
		MaxAge:   int(s.ttl.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil || isHTTPS(r),
	})
	rd := pkce.RD
	if rd == "" {
		rd = "/wall"
	}
	http.Redirect(w, r, rd, http.StatusFound)
}

func (s *Service) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/wall", http.StatusFound)
}

func (s *Service) sessionFromRequest(r *http.Request) (sessionPayload, error) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return sessionPayload{}, err
	}
	return parseSession(s.secret, c.Value)
}

func randToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func isHTTPS(r *http.Request) bool {
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		return true
	}
	return r.TLS != nil
}
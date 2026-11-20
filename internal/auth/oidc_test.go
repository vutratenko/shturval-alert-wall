package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestMiddleware_UnauthorizedAPIRedirectUI(t *testing.T) {
	svc := mustTestService(t)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	h := svc.Wrap(inner)

	t.Run("healthz public", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("healthz: got %d", rr.Code)
		}
	})

	t.Run("api snapshot 401", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/snapshot", nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("snapshot: got %d", rr.Code)
		}
	})

	t.Run("api stream 401", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/stream", nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("stream: got %d", rr.Code)
		}
	})

	t.Run("wall redirects login", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/wall", nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusFound {
			t.Fatalf("wall: got %d", rr.Code)
		}
		if loc := rr.Header().Get("Location"); !strings.HasPrefix(loc, "/oauth2/login") {
			t.Fatalf("location: %q", loc)
		}
	})
}

func TestOIDCFlow_LoginCallbackAccess(t *testing.T) {
	issuer, clientID, clientSecret, key := startFakeOIDC(t)
	redirect := issuer + "/oauth2/callback"
	svc, err := NewService(context.Background(), StaticConfig{
		IssuerURL:    issuer,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirect,
		SessionTTL:   time.Hour,
		CookieSecret: []byte("test-secret-key-32-bytes-long!!"),
	})
	if err != nil {
		t.Fatal(err)
	}
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	h := svc.Wrap(inner)

	loginRR := httptest.NewRecorder()
	h.ServeHTTP(loginRR, httptest.NewRequest(http.MethodGet, "/oauth2/login", nil))
	if loginRR.Code != http.StatusFound {
		t.Fatalf("login: %d", loginRR.Code)
	}
	var pkceC *http.Cookie
	for _, c := range loginRR.Result().Cookies() {
		if c.Name == pkceCookie {
			pkceC = c
		}
	}
	if pkceC == nil {
		t.Fatal("missing pkce cookie")
	}
	raw, _ := base64.RawURLEncoding.DecodeString(pkceC.Value)
	var pkce pkcePayload
	if err := json.Unmarshal(raw, &pkce); err != nil {
		t.Fatal(err)
	}

	_ = key
	cbURL := "/oauth2/callback?code=fake-code&state=" + url.QueryEscape(pkce.State)
	cbReq := httptest.NewRequest(http.MethodGet, cbURL, nil)
	cbReq.AddCookie(pkceC)
	cbRR := httptest.NewRecorder()
	h.ServeHTTP(cbRR, cbReq)
	if cbRR.Code != http.StatusFound {
		t.Fatalf("callback: %d %s", cbRR.Code, cbRR.Body.String())
	}
	var sess *http.Cookie
	for _, c := range cbRR.Result().Cookies() {
		if c.Name == sessionCookieName {
			sess = c
		}
	}
	if sess == nil {
		t.Fatal("no session cookie")
	}

	snapRR := httptest.NewRecorder()
	snapReq := httptest.NewRequest(http.MethodGet, "/api/snapshot", nil)
	snapReq.AddCookie(sess)
	h.ServeHTTP(snapRR, snapReq)
	if snapRR.Code != http.StatusOK {
		t.Fatalf("snapshot after auth: %d", snapRR.Code)
	}
}

func TestLogin_ReusesPKCEState(t *testing.T) {
	svc := mustTestService(t)
	h := svc.Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))

	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/oauth2/login?rd=/wall", nil))
	c := pkceCookieFrom(t, first)
	state1 := stateFromRedirect(t, first)

	second := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth2/login?rd=/wall", nil)
	req.AddCookie(c)
	h.ServeHTTP(second, req)
	if stateFromRedirect(t, second) != state1 {
		t.Fatalf("state rotated: %s vs %s", stateFromRedirect(t, second), state1)
	}
}

func TestCallback_StateMismatchRestartsLogin(t *testing.T) {
	svc := mustTestService(t)
	h := svc.Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))

	login := httptest.NewRecorder()
	h.ServeHTTP(login, httptest.NewRequest(http.MethodGet, "/oauth2/login", nil))
	c := pkceCookieFrom(t, login)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth2/callback?code=x&state=other", nil)
	req.AddCookie(c)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/oauth2/login" {
		t.Fatalf("callback: %d %s", rr.Code, rr.Header().Get("Location"))
	}
}

func pkceCookieFrom(t *testing.T, rr *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rr.Result().Cookies() {
		if c.Name == pkceCookie {
			return c
		}
	}
	t.Fatal("missing pkce cookie")
	return nil
}

func stateFromRedirect(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	loc, err := url.Parse(rr.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	return loc.Query().Get("state")
}

func TestSessionSignParse(t *testing.T) {
	secret := []byte("hmac-secret")
	p := sessionPayload{Sub: "u1", Email: "a@b.c", Exp: time.Now().Add(time.Hour)}
	val, err := signSession(secret, p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseSession(secret, val)
	if err != nil {
		t.Fatal(err)
	}
	if got.Sub != p.Sub || got.Email != p.Email {
		t.Fatalf("got %+v", got)
	}
}

func mustTestService(t *testing.T) *Service {
	t.Helper()
	issuer, clientID, clientSecret, _ := startFakeOIDC(t)
	svc, err := NewService(context.Background(), StaticConfig{
		IssuerURL:    issuer,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  issuer + "/cb",
		SessionTTL:   time.Hour,
		CookieSecret: []byte("test-secret-key-32-bytes-long!!"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func startFakeOIDC(t *testing.T) (issuerURL, clientID, clientSecret string, key *rsa.PrivateKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	clientID = "test-client"
	clientSecret = "test-secret"
	var issuer string
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/auth",
			"token_endpoint":                        issuer + "/token",
			"jwks_uri":                              issuer + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(rsaJWKS(&priv.PublicKey, "test"))
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		_ = r.ParseForm()
		tok := signTestIDToken(priv, issuer, clientID, "user-1", "u@test.local")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "at",
			"token_type":   "Bearer",
			"id_token":     tok,
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	issuer = srv.URL
	return issuer, clientID, clientSecret, priv
}

func rsaJWKS(pub *rsa.PublicKey, kid string) map[string]any {
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
	return map[string]any{
		"keys": []map[string]any{{
			"kty": "RSA", "use": "sig", "alg": "RS256", "kid": kid, "n": n, "e": e,
		}},
	}
}

func signTestIDToken(key *rsa.PrivateKey, issuer, aud, sub, email string) string {
	claims := jwt.MapClaims{
		"iss": issuer, "aud": aud, "sub": sub, "email": email,
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = "test"
	s, err := tok.SignedString(key)
	if err != nil {
		panic(err)
	}
	return s
}
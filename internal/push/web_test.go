package push

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebSender_statusGone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Topic") != "shturval-alerts" {
			t.Errorf("topic %q", r.Header.Get("Topic"))
		}
		w.WriteHeader(http.StatusGone)
	}))
	t.Cleanup(srv.Close)

	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	p256dh, auth := browserKeys(t)
	sender := &WebSender{PublicKey: store.PublicKey(), PrivateKey: store.PrivateKey(), Subject: "mailto:alerts@sion2k.ru"}
	err = sender.Send(context.Background(), Subscription{
		Endpoint: srv.URL,
		Keys:     Keys{P256dh: p256dh, Auth: auth},
	}, []byte(`{"title":"Штурвал","body":"DiskFull","url":"/wall"}`))
	if err != ErrGone {
		t.Fatalf("err %v", err)
	}
}

func browserKeys(t *testing.T) (p256dh, auth string) {
	t.Helper()
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	secret := make([]byte, 16)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(priv.PublicKey().Bytes()), base64.RawURLEncoding.EncodeToString(secret)
}

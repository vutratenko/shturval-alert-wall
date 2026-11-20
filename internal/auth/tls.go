package auth

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
)

func httpClientWithCAB64(caB64 string) (*http.Client, error) {
	if caB64 == "" {
		return http.DefaultClient, nil
	}
	pem, err := base64.StdEncoding.DecodeString(caB64)
	if err != nil {
		return nil, fmt.Errorf("decode CA: %w", err)
	}
	return httpClientWithCAPEM(pem)
}

func httpClientWithCAPEM(pem []byte) (*http.Client, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(pem) {
		return nil, errors.New("failed to append CA certificate")
	}
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
		},
	}, nil
}
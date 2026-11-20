package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var authOidcGVK = schema.GroupVersionKind{
	Group: "auth.shturval.tech", Version: "v1beta1", Kind: "AuthOidcClient",
}

// Bootstrap waits for AuthOidcClient and builds an OIDC Service from cluster resources.
func Bootstrap(ctx context.Context, k8s client.Reader, namespace, clientName, redirectURL string, sessionTTL time.Duration, cookieSecret []byte) (*Service, error) {
	deadline := time.Now().Add(3 * time.Minute)
	var issuer, caB64, secretName string
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(authOidcGVK)
		if err := k8s.Get(ctx, types.NamespacedName{Namespace: namespace, Name: clientName}, obj); err != nil {
			if apierrors.IsNotFound(err) {
				time.Sleep(3 * time.Second)
				continue
			}
			return nil, err
		}
		status, _, _ := unstructured.NestedMap(obj.Object, "status")
		if status == nil {
			time.Sleep(3 * time.Second)
			continue
		}
		ready, _, _ := unstructured.NestedBool(status, "ready")
		if !ready {
			time.Sleep(3 * time.Second)
			continue
		}
		clientInfo, _, _ := unstructured.NestedMap(status, "clientInfo")
		if clientInfo != nil {
			issuer, _, _ = unstructured.NestedString(clientInfo, "issuerUrl")
			caB64, _, _ = unstructured.NestedString(clientInfo, "caFileBase64")
		}
		ref, _, _ := unstructured.NestedMap(status, "clientSecretRef")
		if ref != nil {
			secretName, _, _ = unstructured.NestedString(ref, "name")
		}
		if issuer != "" && secretName != "" {
			break
		}
		time.Sleep(3 * time.Second)
	}
	if issuer == "" || secretName == "" {
		return nil, fmt.Errorf("AuthOidcClient %q not ready in namespace %q", clientName, namespace)
	}
	sec := &corev1.Secret{}
	if err := k8s.Get(ctx, types.NamespacedName{Namespace: namespace, Name: secretName}, sec); err != nil {
		return nil, fmt.Errorf("read oidc secret: %w", err)
	}
	clientID, err := secretData(sec, "client_id")
	if err != nil {
		return nil, err
	}
	clientSecret, err := secretData(sec, "client_secret")
	if err != nil {
		return nil, err
	}
	httpClient, err := httpClientWithCAB64(caB64)
	if err != nil {
		return nil, err
	}
	return NewService(ctx, StaticConfig{
		IssuerURL:    issuer,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		SessionTTL:   sessionTTL,
		CookieSecret: cookieSecret,
		HTTPClient:   httpClient,
	})
}

func secretData(sec *corev1.Secret, key string) (string, error) {
	raw, ok := sec.Data[key]
	if !ok {
		return "", fmt.Errorf("secret %q missing key %q", sec.Name, key)
	}
	if len(raw) == 0 {
		return "", fmt.Errorf("secret %q key %q is empty", sec.Name, key)
	}
	// Operator stores plain bytes; tolerate base64-looking values.
	if s := string(raw); isPrintableASCII(s) {
		return s, nil
	}
	return string(raw), nil
}

func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}

// CookieSecretFromEnv decodes AUTH_COOKIE_SECRET (base64), or returns nil.
func CookieSecretFromEnv(b64 string) ([]byte, error) {
	if b64 == "" {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(b64)
}
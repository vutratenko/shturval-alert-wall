package auth

import "strings"

func isPublicPath(path string) bool {
	switch path {
	case "/api/healthz", "/metrics":
		return true
	}
	if strings.HasPrefix(path, "/oauth2/") {
		return true
	}
	return false
}

func isAPIPath(path string) bool {
	return strings.HasPrefix(path, "/api/")
}
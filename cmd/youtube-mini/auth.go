package main

import (
	"encoding/base64"
	"net/http"
	"strings"
)

var adminUser = "admin"
var adminPass = "qwerty1234" // change

func checkAuth(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return false
	}

	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return false
	}

	decoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return false
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return false
	}

	username, password := parts[0], parts[1]
	return username == adminUser && password == adminPass
}

func requireAuth(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="YouTube Mini Admin"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

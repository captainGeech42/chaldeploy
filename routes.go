package main

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"log"
)

// GET /healthcheck
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("app good to go"))
}

// POST /api/auth
// takes the auth url/login token, and gets an auth token for the rCTF api
func clientAuth(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("error handling client auth, couldn't read body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bodyStr := string(body)
	parts := strings.Split(bodyStr, "/login?token=")
	loginTokenEncoded := parts[len(parts)-1]

	loginToken, err := url.QueryUnescape(loginTokenEncoded)
	if err != nil {
		log.Printf("error handling client auth, couldn't decode login token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	authToken, err := authToRctf(loginToken)
	if err != nil {
		log.Printf("error handling client auth, couldn't auth to rCTF: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if authToken == "" {
		w.WriteHeader(http.StatusForbidden)
	} else {
		w.Write([]byte(authToken))
	}
}

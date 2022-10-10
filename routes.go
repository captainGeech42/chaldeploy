package main

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"log"

	"github.com/gorilla/sessions"
)

// GET /healthcheck
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("app good to go"))
}

// POST /api/auth
// Takes the auth url/login token, and gets an auth token for the rCTF api
// Returns back the team name and 200 if successful, otherwise 403/500+
func clientAuth(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
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
		return
	}

	// have a valid auth token, get team info
	userInfo, err := getUserInfo(authToken)
	if err != nil {
		log.Printf("error handling client auth, couldn't get user info from rCTF: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// save the team data to the user's session
	s.Values["teamName"] = userInfo.TeamName
	s.Values["id"] = userInfo.Id
	s.Values["authToken"] = authToken
	s.Save(r, w)

	// send back the team name
	w.Write([]byte(userInfo.TeamName))
}

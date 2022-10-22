package main

import (
	"encoding/json"
	// deliberately using this instead of html/template to leave html comments in more easily.
	// templated data is not user controlled
	"text/template"

	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"log"

	"github.com/gorilla/sessions"
)

var created bool = false

// don't flame me, i'm lazy
var cachedIndex = ""
var cachedIndexLock sync.Mutex

func indexPage(w http.ResponseWriter, r *http.Request) {
	if config == nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("indexPage was called before config was set, can't render template")
	}

	// check if the index has been rendered yet
	if cachedIndex == "" {
		log.Println("need to render the index page")

		// index hasn't been rendered yet. lock the resource and render it
		cachedIndexLock.Lock()
		defer cachedIndexLock.Unlock()

		if cachedIndex == "" {
			// why do we check it again? good question, smart reader who is smarter than the dingdong who wrote this
			// method! i think its possible for a second caller of this function to get into this code path by
			// getting blocked waiting for the lock, and we don't want them to re-render the template if they don't
			// need to. so, allow them to bail out and prevent re-rendering. stupid? yes. works? probably. need it?
			// not a clue. have fun.

			log.Println("actually rendering the index page")

			t, err := template.ParseFiles("templates/index.html")
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Printf("failed to parse index template: %v\n", err)
				return
			}

			sb := &strings.Builder{}
			err = t.Execute(sb, config)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Printf("failed to render index template: %v\n", err)
				return
			}

			cachedIndex = sb.String()
		} else {
			log.Println("index page got rendered for me, yeet")
		}
	}

	w.Write([]byte(cachedIndex))
}

// GET /healthcheck
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("app good to go"))
}

// POST /api/auth
// Takes the auth url/login token, and gets an auth token for the rCTF api
// Returns back the team name and 200 if successful, otherwise 403/500+
func authRequest(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
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
	if err = s.Save(r, w); err != nil {
		log.Printf("error handling client auth, couldn't save the session: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// send back the team name
	w.Write([]byte(userInfo.TeamName))
}

type StatusResponse struct {
	State string `json:"state"` // "active" || "inactive"
	Host  string `json:"host,omitempty"`
	// ExpTime
}

// GET /api/status
// Get the status of the team's deployment
func statusRequest(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	// make sure the session is valid
	if _, exists := s.Values["id"]; s.IsNew || !exists {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// TODO: check k8s for instance

	var resp StatusResponse

	if created {
		resp = StatusResponse{State: "active", Host: "1.2.3.4:8989"}
	} else {
		resp = StatusResponse{State: "inactive"}
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		log.Printf("error handling status request, couldn't marshal response data: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(respBytes)
}

type CreateInstanceResponse struct {
	Host string `json:"host"` // host:port string
	// ExpTime
}

// POST /api/create
// Create a deployment instance for the team
func createInstanceRequest(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	// make sure the session is valid
	if _, exists := s.Values["id"]; s.IsNew || !exists {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	log.Printf("Deploying instance for %s (ID: %s)\n", s.Values["teamName"], s.Values["id"])

	// create the deployment
	cxn, err := im.CreateDeployment(s.Values["teamName"].(string), s.Values["id"].(string))
	if err != nil {
		log.Printf("couldn't create a deployment for %s: %v", s.Values["teamName"], err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp := CreateInstanceResponse{Host: cxn}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		log.Printf("error handling create instance request, couldn't marshal response data: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-type", "application/json")
	w.Write(respBytes)
}

// POST /api/extend
// Extend the timeout for a deployment instance
// Response on 200 is the new expiration timestamp
func extendInstanceRequest(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	// make sure the session is valid
	if _, exists := s.Values["id"]; s.IsNew || !exists {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	log.Printf("Extending instance for %s (ID: %s)\n", s.Values["teamName"], s.Values["id"])

	// TODO: extend instance and update memcache

	w.Header().Add("Content-type", "text/plain")
	w.Write([]byte("2022-01-01 12:34:56"))
}

// POST /api/destroy
// Destroy a deployment instance
// 200 means successfully destroy
func destroyInstanceRequest(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	// make sure the session is valid
	if _, exists := s.Values["id"]; s.IsNew || !exists {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	log.Printf("Destroying instance for %s (ID: %s)\n", s.Values["teamName"], s.Values["id"])

	// TODO: destroy instance and update memcache

	created = false

	w.WriteHeader(http.StatusOK)
}

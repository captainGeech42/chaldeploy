package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

var RCTF_SERVER = "https://2021.redpwn.net"

var store = sessions.NewCookieStore([]byte(os.Getenv("CHALDEPLOY_SESSION_KEY")))

// Log the incoming requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// don't log healthcheck b/c i don't care
		if r.RequestURI == "/healthcheck" {
			return
		}

		log.Printf("%s request from %s to %s\n", r.Method, r.RemoteAddr, r.RequestURI)

		next.ServeHTTP(w, r)
	})
}

// custom http.Handler that adds a session parameter for router handlers to leverage
type sessionHandler func(w http.ResponseWriter, r *http.Request, s *sessions.Session)

func (h sessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s, _ := store.Get(r, "session")

	h(w, r, s)
}

/*
can do an in memory map to track team info and deployment status. on startup, go to k8s and populate the map
this means that there can only be one instance of this running, and also have to use locks on the map
easier than doing a db though
*/

func main() {
	// deployApp("OSUSEC")

	router := mux.NewRouter()

	store.Options.SameSite = http.SameSiteStrictMode

	router.Use(loggingMiddleware)
	router.HandleFunc("/healthcheck", healthCheck)
	router.Path("/api/auth").Handler(sessionHandler(authRequest)).Methods("POST")
	router.Path("/api/status").Handler(sessionHandler(statusRequest))
	router.Path("/api/create").Handler(sessionHandler(createInstanceRequest)).Methods("POST")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	log.Println("starting server on port 5050")
	log.Fatalln(http.ListenAndServe(":5050", router))
}

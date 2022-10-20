package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

// globals
var RCTF_SERVER = ""
var store *sessions.CookieStore

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
	// load config
	config, err := loadConfig()
	if err != nil {
		log.Fatalln(err)
	}

	// initialize router
	router := mux.NewRouter()

	// initialize session store
	if sessKeyLen := len(config.SessionKey); !Contains([]int{32, 64}, sessKeyLen) {
		log.Fatalf("the session key is an invalid length: %d (must be 32 or 64)\n", sessKeyLen)
	}
	store = sessions.NewCookieStore([]byte(config.SessionKey))
	store.Options.SameSite = http.SameSiteStrictMode

	// register routes
	router.Use(loggingMiddleware)
	router.HandleFunc("/healthcheck", healthCheck)
	router.Path("/api/auth").Handler(sessionHandler(authRequest)).Methods("POST")
	router.Path("/api/status").Handler(sessionHandler(statusRequest))
	router.Path("/api/create").Handler(sessionHandler(createInstanceRequest)).Methods("POST")
	router.Path("/api/extend").Handler(sessionHandler(extendInstanceRequest)).Methods("POST")
	router.Path("/api/destroy").Handler(sessionHandler(destroyInstanceRequest)).Methods("POST")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	// start the server
	log.Println("starting server on port 5050")
	log.Fatalln(http.ListenAndServe(":5050", router))
}

package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

var RCTF_SERVER = "https://2021.redpwn.net"

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

/*
can do an in memory map to track team info and deployment status. on startup, go to k8s and populate the map
this means that there can only be one instance of this running, and also have to use locks on the map
easier than doing a db though
*/

func main() {
	// deployApp("OSUSEC")

	router := mux.NewRouter()

	router.Use(loggingMiddleware)
	router.HandleFunc("/healthcheck", healthCheck)
	router.HandleFunc("/api/auth", clientAuth).Methods("POST")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	log.Println("starting server on port 5050")
	log.Fatalln(http.ListenAndServe(":5050", router))
}

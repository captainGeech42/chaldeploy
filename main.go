package main

import (
	"log"
	"net/http"
)

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("app good to go"))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// don't log healthcheck b/c i don't care
		if r.RequestURI == "/healthcheck" {
			return
		}

		log.Printf("request from %s to %s\n", r.RemoteAddr, r.RequestURI)

		next.ServeHTTP(w, r)
	})
}

func main() {
	// deployApp("OSUSEC")

	mux := http.NewServeMux()

	mux.HandleFunc("/healthcheck", healthCheck)
	mux.Handle("/", http.FileServer(http.Dir("./static/")))

	log.Println("starting server on port 5050")
	log.Fatalln(http.ListenAndServe(":5050", loggingMiddleware(mux)))
}

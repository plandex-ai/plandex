package server

import (
 "log"
 "net/http"
 "time"
)

func init() { log.Printf("Server starting at %s...", time.Now()) }

func handler(w http.ResponseWriter, r *http.Request) { 
    w.Write([]byte("Hello, world!")) 
    log.Printf("Request: %s %s", r.Method, r.RequestURI)
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("API is live"))
}
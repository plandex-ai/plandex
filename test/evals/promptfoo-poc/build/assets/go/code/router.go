pdx-1: // Package router implements a simple HTTP request router
pdx-2: package router
pdx-3: 
pdx-4: import (
pdx-5: 	"net/http"
pdx-6: )
pdx-7: 
pdx-8: type HandlerFn func(http.ResponseWriter, *http.Request)
pdx-9: 
pdx-10: type Router struct {
pdx-11: 	paths map[string]HandlerFn
pdx-12: }
pdx-13: 
pdx-14: // New creates a new Router
pdx-15: func New() *Router {
pdx-16: 	return &Router{
pdx-17: 		paths: make(map[string]HandlerFn),
pdx-18: 	}
pdx-19: }
pdx-20: 
pdx-21: // Register registers a new route with a handler
pdx-22: func (r *Router) Register(path string, handler HandlerFn) {
pdx-23: 	r.paths[path] = handler
pdx-24: }
pdx-25: 
pdx-26: // ServeHTTP implements http.Handler
pdx-27: func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
pdx-28: 	if handler, ok := r.paths[req.URL.Path]; ok {
pdx-29: 		handler(w, req)
pdx-30: 	} else {
pdx-31: 		w.WriteHeader(http.StatusNotFound)
pdx-32: 		w.Write([]byte("404 - Not Found"))
pdx-33: 	}
pdx-34: }

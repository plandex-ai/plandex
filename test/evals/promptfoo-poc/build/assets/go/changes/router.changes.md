// Added logging

func (r *Router) Register(path string, handler HandlerFn) {

log.Printf("Registering path: %s", path)

r.paths[path] = handler

}

// Added middleware support

type MiddlewareFn func(HandlerFn) HandlerFn

var middlewares []MiddlewareFn

// Use adds a new middleware

func Use(middleware MiddlewareFn) {

middlewares = append(middlewares, middleware)

}

// wrappedHandler wraps the registered handlers with the applied middlewares

func wrappedHandler(handler HandlerFn) HandlerFn {

for _, middleware := range middlewares {

handler = middleware(handler)

}

return handler

}

// Apply middlewares to handlers

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {

if handler, ok := r.paths[req.URL.Path]; ok {

handler = wrappedHandler(handler)

handler(w, req)

} else {

w.WriteHeader(http.StatusNotFound)

w.Write([]byte("404 - Not Found"))

}

}


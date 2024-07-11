pdx-1: package middleware 
pdx-2: 
pdx-3: // Middleware defines a function to process middleware
pdx-4: type Middleware func(next Handler) Handler
pdx-5: 
pdx-6: // Handler defines the request handler used by the middleware
pdx-7: type Handler func() error
pdx-8: 
pdx-9: // Chain chains the middleware functions
pdx-10: func Chain(middlewares ...Middleware) Middleware {
pdx-11: 	return func(next Handler) Handler {
pdx-12: 		for i := len(middlewares) - 1; i >= 0; i-- {
pdx-13: 			middleware := middlewares[i]
pdx-14: 			next = middleware(next)
pdx-15: 		}
pdx-16: 		return next
pdx-17: 	}
pdx-18: }

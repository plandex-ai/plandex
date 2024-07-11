// Modified the Chain function to accept context for more flexible middleware handling

import "context"

// Middleware now uses context

type Middleware func(ctx context.Context, next Handler) Handler

// Handler now accepts context

type Handler func(ctx context.Context) error

// Chain now also accepts context and passes it down the middleware chain

func Chain(middlewares ...Middleware) Middleware {

return func(ctx context.Context, next Handler) Handler {

for i := len(middlewares) - 1; i >= 0; i-- {

middleware := middlewares[i]

next = middleware(ctx, next)

}

return next

}

}


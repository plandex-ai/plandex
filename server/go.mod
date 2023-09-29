module plandex-server

go 1.21.0

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/google/uuid v1.3.1
	github.com/gorilla/mux v1.8.0
	github.com/plandex/plandex/shared v0.0.0-00010101000000-000000000000
	github.com/sashabaranov/go-openai v1.15.1
)

replace github.com/plandex/plandex/shared => ../shared

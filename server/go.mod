module plandex-server

go 1.21.3

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/google/uuid v1.4.0
	github.com/gorilla/mux v1.8.1
	github.com/plandex/plandex/shared v0.0.0-00010101000000-000000000000
	github.com/sashabaranov/go-openai v1.17.5
)

require (
	github.com/dlclark/regexp2 v1.10.0 // indirect
	github.com/pkoukk/tiktoken-go v0.1.6 // indirect
)

require github.com/looplab/fsm v1.0.1 // indirect

replace github.com/plandex/plandex/shared => ../shared

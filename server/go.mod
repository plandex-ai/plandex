module plandex-server

go 1.21.3

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/google/uuid v1.3.1
	github.com/gorilla/mux v1.8.0
	github.com/plandex/plandex/shared v0.0.0-00010101000000-000000000000
	github.com/sashabaranov/go-openai v1.15.1
)

require (
	github.com/dlclark/regexp2 v1.10.0 // indirect
	github.com/pkoukk/tiktoken-go v0.1.6 // indirect
)

require (
	github.com/drhodes/golorem v0.0.0-20220328165741-da82e5b29246
	github.com/looplab/fsm v1.0.1 // indirect
)

replace github.com/plandex/plandex/shared => ../shared

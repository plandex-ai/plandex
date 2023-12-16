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
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/pkoukk/tiktoken-go v0.1.6 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	golang.org/x/sys v0.14.0 // indirect
)

require (
	github.com/fatih/color v1.16.0
	github.com/golang-migrate/migrate/v4 v4.16.2
	github.com/jmoiron/sqlx v1.3.5
	github.com/lib/pq v1.10.9
	github.com/looplab/fsm v1.0.1 // indirect
)

replace github.com/plandex/plandex/shared => ../shared

module plandex-server

go 1.21.0

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/google/uuid v1.3.1
	github.com/gorilla/mux v1.8.0
	github.com/plandex/plandex/shared v0.0.0-00010101000000-000000000000
	github.com/sashabaranov/go-openai v1.15.1
)

require (
	github.com/dlclark/regexp2 v1.10.0 // indirect
	github.com/fatih/color v1.9.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/githubnemo/CompileDaemon v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.11 // indirect
	github.com/pkoukk/tiktoken-go v0.1.6 // indirect
	github.com/radovskyb/watcher v1.0.7 // indirect
	golang.org/x/sys v0.0.0-20191026070338-33540a1f6037 // indirect
)

require (
	github.com/drhodes/golorem v0.0.0-20220328165741-da82e5b29246
	github.com/looplab/fsm v1.0.1 // indirect
)

replace github.com/plandex/plandex/shared => ../shared

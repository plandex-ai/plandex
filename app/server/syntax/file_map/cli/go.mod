module mapper

go 1.23.3

replace plandex-server => ../../../

replace plandex-shared => ../../../../shared

replace plandex => ../../../../cli

require (
	plandex v0.0.0-00010101000000-000000000000
	plandex-server v0.0.0-00010101000000-000000000000
	plandex-shared v0.0.0-00010101000000-000000000000
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/pkoukk/tiktoken-go v0.1.7 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06 // indirect
	github.com/sashabaranov/go-openai v1.36.1 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/smacker/go-tree-sitter v0.0.0-20240827094217-dd81d9e9be82 // indirect
	golang.org/x/image v0.23.0 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
)

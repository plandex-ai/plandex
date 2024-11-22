module mapper

go 1.23.3

replace plandex-server => ../../

replace github.com/plandex/plandex/shared => ../../../shared

require plandex-server v0.0.0-00010101000000-000000000000

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/pkoukk/tiktoken-go v0.1.7 // indirect
	github.com/plandex/plandex/shared v0.0.0-00010101000000-000000000000 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sashabaranov/go-openai v1.35.6 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/smacker/go-tree-sitter v0.0.0-20240625050157-a31a98a7c0f6 // indirect
	golang.org/x/image v0.22.0 // indirect
)

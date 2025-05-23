module plandex-app

go 1.23.9

require (
	github.com/spf13/cobra v1.9.1
	plandex-shared v0.0.0-00010101000000-000000000000
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/pkoukk/tiktoken-go v0.1.7 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sashabaranov/go-openai v1.38.1 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	golang.org/x/image v0.25.0 // indirect
)

replace plandex-shared => ./app/shared

replace plandex-server/types => ./app/server/types

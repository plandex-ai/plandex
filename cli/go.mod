module plandex

go 1.21.0

require github.com/spf13/cobra v1.7.0

require (
	github.com/andybalholm/cascadia v1.3.1 // indirect
	github.com/dlclark/regexp2 v1.10.0 // indirect
	github.com/fatih/color v1.7.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/mattn/go-colorable v0.1.2 // indirect
	github.com/mattn/go-isatty v0.0.8 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/term v0.5.0 // indirect
)

require (
	github.com/PuerkitoBio/goquery v1.8.1
	github.com/briandowns/spinner v1.23.0
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pkoukk/tiktoken-go v0.1.5
	github.com/plandex/plandex/shared v0.0.0-00010101000000-000000000000
	github.com/sashabaranov/go-openai v1.15.1
	github.com/spf13/pflag v1.0.5 // indirect
)

replace github.com/plandex/plandex/shared => ../shared

module plandex-server

go 1.23.3

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/pkg/errors v0.9.1
	github.com/sashabaranov/go-openai v1.38.1
	plandex-shared v0.0.0-00010101000000-000000000000
)

require (
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/go-toast/toast v0.0.0-20190211030409-01e6764cf0a4 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/pkoukk/tiktoken-go v0.1.7 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/tadvi/systray v0.0.0-20190226123456-11a2b8fa57af // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/image v0.25.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/atotto/clipboard v0.1.4
	github.com/aws/aws-sdk-go v1.55.6
	github.com/fatih/color v1.18.0
	github.com/gen2brain/beeep v0.0.0-20240516210008-9c006672e7f4
	github.com/golang-migrate/migrate/v4 v4.18.2
	github.com/jmoiron/sqlx v1.4.0
	github.com/lib/pq v1.10.9
	github.com/smacker/go-tree-sitter v0.0.0-20240827094217-dd81d9e9be82
	github.com/stretchr/testify v1.10.0
	golang.org/x/net v0.34.0
)

replace plandex-shared => ../shared

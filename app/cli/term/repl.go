package term

import "os"

var IsRepl = os.Getenv("PLANDEX_REPL") != ""

func SetIsRepl(value bool) {
	IsRepl = value
}

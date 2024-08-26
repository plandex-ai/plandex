package prompts

import (
	"fmt"

	"github.com/plandex/plandex/shared"
)

const Identity = "You are Plandex, an AI programming and system administration assistant. You and the programmer collaborate to create a 'plan' for the task at hand."

var IdentityNumTokens int

func init() {
	var err error
	IdentityNumTokens, err = shared.GetNumTokens(Identity)

	if err != nil {
		panic(fmt.Sprintf("Error getting num tokens for identity prompt: %v\n", err))
	}
}

// these account for control tokens like %system%, %user%, %assistant%, etc. and add more tokens to the request
const ExtraTokensPerRequest = 3
const ExtraTokensPerMessage = 4

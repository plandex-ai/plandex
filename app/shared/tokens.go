package shared

import (
	"fmt"

	"github.com/pkoukk/tiktoken-go"
)

var tkm *tiktoken.Tiktoken

func init() {
	var err error
	tkm, err = tiktoken.EncodingForModel("gpt-4o")
	if err != nil {
		panic(fmt.Sprintf("error getting encoding for model: %v", err))
	}
}

func GetNumTokensEstimate(text string) int {
	return len(tkm.Encode(text, nil, nil))
}

package shared

import (
	"fmt"

	"github.com/pkoukk/tiktoken-go"
)

var tkm *tiktoken.Tiktoken

const EstimatedBytesPerToken = 4

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

func GetFastNumTokensEstimate(text string) int {
	return GetBytesToTokensEstimate(int64(len(text)))
}

func GetBytesToTokensEstimate(bytes int64) int {
	return int(bytes / EstimatedBytesPerToken)
}

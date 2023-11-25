package shared

import (
	"fmt"

	"github.com/pkoukk/tiktoken-go"
)

var MaxTokens int = 120000
var MaxContextTokens int = 50000
var MaxConvoTokens int = 20000

func GetNumTokens(text string) (int, error) {
	tkm, err := tiktoken.EncodingForModel("gpt-4")
	if err != nil {
		err = fmt.Errorf("error getting encoding for model: %v", err)
		return 0, err
	}
	return len(tkm.Encode(text, nil, nil)), nil
}

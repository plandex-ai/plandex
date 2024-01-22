package shared

import (
	"fmt"

	"github.com/pkoukk/tiktoken-go"
)

var MaxTokens int = 7000        //120000
var MaxContextTokens int = 5000 //50000
var MaxConvoTokens int = 2000   //20000

func GetNumTokens(text string) (int, error) {
	tkm, err := tiktoken.EncodingForModel("gpt-4")
	if err != nil {
		err = fmt.Errorf("error getting encoding for model: %v", err)
		return 0, err
	}
	return len(tkm.Encode(text, nil, nil)), nil
}

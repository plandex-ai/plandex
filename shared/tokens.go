package shared

import (
	"fmt"
	"log"

	"github.com/pkoukk/tiktoken-go"
)

var MaxTokens int = 120000
var MaxContextTokens int = 50000
var MaxConvoTokens int = 20000

func GetNumTokens(text string) (numTokens int) {
	tkm, err := tiktoken.EncodingForModel("gpt-4")
	if err != nil {
		err = fmt.Errorf("encoding for model: %v", err)
		log.Println(err)
		return
	}
	return len(tkm.Encode(text, nil, nil))
}

package shared

import (
	"fmt"
	"log"

	"github.com/pkoukk/tiktoken-go"
)

const MaxTokens int = 8000

func GetNumTokens(text string) (numTokens int) {
	tkm, err := tiktoken.EncodingForModel("gpt-4")
	if err != nil {
		err = fmt.Errorf("encoding for model: %v", err)
		log.Println(err)
		return
	}

	return len(tkm.Encode(text, nil, nil))
}

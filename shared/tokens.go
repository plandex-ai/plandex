package shared

import (
	"fmt"
	"log"

	"github.com/pkoukk/tiktoken-go"
)

const MaxTokens int = 7500        // leave room for the model ouput
const MaxContextTokens int = 6500 // leave room for the conversation history

func GetNumTokens(text string) (numTokens int) {
	tkm, err := tiktoken.EncodingForModel("gpt-4")
	if err != nil {
		err = fmt.Errorf("encoding for model: %v", err)
		log.Println(err)
		return
	}
	return len(tkm.Encode(text, nil, nil))
}

package lib

import (
	"fmt"
	"log"

	"github.com/pkoukk/tiktoken-go"
)

const MaxTokens uint32 = 8000

func GetNumTokens(text string) (numTokens uint32) {
	tkm, err := tiktoken.EncodingForModel("gpt-4")
	if err != nil {
		err = fmt.Errorf("encoding for model: %v", err)
		log.Println(err)
		return
	}

	return uint32(len(tkm.Encode(text, nil, nil)))
}

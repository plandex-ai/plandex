package prompts

import (
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const SysFileName = "You are an AI namer that creates a valid filename for the content. Content can be any text, including programs/code, documentation, websites, and more. Most text will be related to software development. Call the 'nameFile' function with a valid JSON object that includes the 'fileName' key. Only call the 'nameFile' function in your response. Don't call any other function."

var FileNameFn = openai.FunctionDefinition{
	Name: "nameFile",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"fileName": {
				Type:        jsonschema.String,
				Description: "A *short* file name for the text based on the content. Use dashes as word separators. No spaces or special characters. **2-3 words max**.",
			},
		},
		Required: []string{"fileName"},
	},
}

func GetFileNamePrompt(text string) string {
	return SysFileName + "\n\nText:\n" + text
}

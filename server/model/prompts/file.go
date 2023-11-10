package prompts

import (
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const SysWriteFile = `
[YOUR INSTRUCTIONS]

You are an AI plan builder. You apply changes from a plan to a given code file. Use 'writeEntireFile' function to write the full content of the file as raw text, including any updates from the previous response, to the file. Call 'writeEntireFile' with the entire updated file. Don't include any placeholders or references to the original file.

Again, make sure the output represents the ENTIRE FILE with all your suggested modifications from the plan applied to it. You MUST NOT include placeholders like '... Rest of the function' or '... Rest of the imports' or '... Rest of the file'. You MUST NOT include references to the original file like '... Rest of the file from the previous response'. You MUST output the ENTIRE file WITHOUT EXCEPTION even if it causes you to generate a large number of tokens. The entire file MUST be output as raw text. This is VERY IMPORTANT, as your output will replace the original file from the context. If any sections of the original file are missing in your output, the user's code will be broken. Please output the ENTIRE FILE with all suggested modifications applied to it.

[END INSTRUCTIONS]`

var WriteFileFn = openai.FunctionDefinition{
	Name: "writeEntireFile",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"content": {
				Type:        jsonschema.String,
				Description: "The full content of the file, including any updates from the previous response, as raw text",
			},
		},
		Required: []string{"content"},
	},
}

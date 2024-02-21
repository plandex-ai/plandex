package prompts

import (
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const SysDescribe = "You are an AI parser. You turn an AI's plan for a programming task into a structured description. Call the 'describePlan' function with a valid JSON object that includes the 'commitMsg' key. 'commitMsg' should be a good, succinct commit message for the changes proposed."

var DescribePlanFn = openai.FunctionDefinition{
	Name: "describePlan",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"commitMsg": {
				Type: jsonschema.String,
			},
		},
		Required: []string{"commitMsg"},
	},
}

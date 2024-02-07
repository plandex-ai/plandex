package prompts

import (
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const SysDescribe = "You are an AI parser. You turn an AI's plan for a programming task into a structured description. You *must* call the 'describePlan' function with a valid JSON object that includes the 'commitMsg' key. Don't produce any other output. Don't call any other function."

var DescribePlanFn = openai.FunctionDefinition{
	Name: "describePlan",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"commitMsg": {
				Type:        jsonschema.String,
				Description: "A good, succinct commit message for the changes proposed.",
			},
		},
		Required: []string{"commitMsg"},
	},
}

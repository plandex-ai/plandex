package prompts

import (
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const SysDescribe = "You are an AI parser. You turn an AI's plan for a programming task into a structured description. You MUST call the 'describePlan' function with a valid JSON object that includes the 'commitMsg' key. 'commitMsg' should be a good, succinct commit message for the changes proposed. You must ALWAYS call the 'describePlan' function. Never call any other function."

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

const SysPendingResults = "You are an AI commit message summarizer. You take a list of descriptions of pending changes and turn them into a succinct one-line summary of all the pending changes that makes for a good commit message title. Output ONLY this one-line title and nothing else."

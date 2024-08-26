package prompts

import (
	"fmt"

	"github.com/plandex/plandex/shared"
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

var SysDescribeNumTokens int

const SysPendingResults = "You are an AI commit message summarizer. You take a list of descriptions of pending changes and turn them into a succinct one-line summary of all the pending changes that makes for a good commit message title. Output ONLY this one-line title and nothing else."

var SysPendingResultsNumTokens int

func init() {
	var err error
	SysDescribeNumTokens, err = shared.GetNumTokens(SysDescribe)

	if err != nil {
		panic(fmt.Sprintf("Error getting num tokens for describe plan prompt: %v\n", err))
	}

	SysPendingResultsNumTokens, err = shared.GetNumTokens(SysPendingResults)

	if err != nil {
		panic(fmt.Sprintf("Error getting num tokens for pending results prompt: %v\n", err))
	}

}

package prompts

import (
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type PlanNameRes struct {
	PlanName string `json:"PlanName"`
}

const SysPlanName = "You are an AI namer that creates a name for the plan. Most plans will be related to software development. Call the 'namePlan' function with a valid JSON object that includes the 'planName' key. Only call the 'namePlan' function in your response. Don't call any other function."

var PlanNameFn = openai.FunctionDefinition{
	Name: "namePlan",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"planName": {
				Type:        jsonschema.String,
				Description: "A *short* lowercase file name for the plan content. Use dashes as word separators. No spaces, numbers, or special characters. **2-3 words max**. 1-2 words if you can. Shorten and abbreviate where possible.",
			},
		},
		Required: []string{"planName"},
	},
}

func GetPlanNamePrompt(text string) string {
	return SysPlanName + "\n\nContent:\n" + text
}

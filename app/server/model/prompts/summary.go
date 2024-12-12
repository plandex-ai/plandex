package prompts

import (
	"fmt"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const SysShortSummary = "You are an AI summarizer that summarizes text, including programs, documentation, websites, and more. Most text will be related to software development. You produce a brief summary of the text. A few sentences at most."

var ShortSummaryFn = openai.FunctionDefinition{
	Name: "summarize",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"summary": {
				Type: jsonschema.String,
			},
		},
		Required: []string{"summary"},
	},
}

func GetShortSummaryPrompt(text string) string {
	return `Please summarize the text below using the 'summarize' function. Call 'summarize' with valid JSON that includes the 'summary' key. You must ALWAYS call the 'summarize' function in your reponse. Don't call any other function.
	
	Text:
					
	` + text
}

const PlanSummary = `
You are an AI summarizer that summarizes the conversation so far. The conversation so far is a plan to complete one or more programming tasks for a user. This conversation may begin with an existing summary of the plan.

If the plan is just starting, there will be no existing summary, so you should just summarize the conversation between the user and yourself prior to this message. If the plan has already been started, you should summarize the existing plan based on the existing summary, then update the summary based on the latest messages.

Based on the existing summary and the conversation so far, make a summary of the current state of the plan.

Do not include any heading or title for the summary. Just start with the summary of the plan.

- Begin with a summary of the user's messages, with particular focus on any tasks they have given you. Your summary of the tasks should reflect the latest version of each task--if they have changed over time, summarize the latest state of each task that was given and omit anything that is now obsolete. Condense this information as much as possible while still being clear and retaining the meaning of the original messages.

- Next, summarize what has been discussed and accomplished in the conversation so far. This should include:
  - Key decisions that have been made
  - Major changes or updates to the plan
  - Any significant challenges or considerations that have been identified
  - Important requirements or constraints that have been established

- Last, summarize what has been done in the latest messages and any next steps or action items that have been discussed.

- Do not include code in the summary. Explain in words what has been done and what needs to be done.

- Treat the summary as *append-only*. Keep as much information as possible from the existing summary and add the new information from the latest messages. The summary is meant to be a record of the entire plan as it evolves over time.

Output only the summary of the current state of the plan and nothing else.
`

var PlanSummaryNumTokens int

func init() {
	var err error
	PlanSummaryNumTokens, err = shared.GetNumTokens(PlanSummary)

	if err != nil {
		panic(fmt.Sprintf("Error getting num tokens for plan summary prompt: %v\n", err))
	}
}

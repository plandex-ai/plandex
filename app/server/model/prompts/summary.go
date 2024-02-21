package prompts

import (
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
	return `
						 Please summarize the text below using the 'summarize' function. Call 'summarize' with valid JSON that includes the 'summary' key. Only call the 'summarize' function in your reponse. Don't call any other function.

						 Text:

					` + text
}

const PlanSummary = `
Based on the conversation so far, make a summary of the current state of the plan. 

- It should begin with a summary of the user's messages, with particular focus any tasks they have given you. Your summary of the tasks should reflect the latest version of each task--if they have changed over time, summarize the latest state of each task that was given and omit anything that is now obsolete. Condense this information as much as possible while still being clear and retaining the meaning of the original messages. If the user has sent messages like 'continue to the next step of the plan' that don't contain any new information relevant to the plan, you should omit them.

- Next, if the plan includes a statement from the assistant to the effect of "I will break this large task into subtasks" and the plan has been broken down into subtasks, include those subtasks in the summary as a numbered list. Condense these as much as possible while still being clear and retaining the meaning of each subtask. At the end of the list, state which subtask is currently being worked on (unless they are all finished, in which case state that they are all finished).

- Last, summarize the latest version of the plan and any changes you have suggested. If the some of the older changes have been overridden by newer changes, you should only include the newest changes and omit the older ones. 

- If your responses include code blocks labelled with file paths, include the latest state of your modifications to each file's code and label them with file paths in the same format as the original messages. Do not make new changes to the plan or to suggested code changes in your summary.

- If code blocks can be converted to a textual summary without losing meaning or precision, summarize them. Otherwise, leave them as code.

- If part of the plan has already been summarized, update the summary based on the latest messages while continuing the follow the above guidelines.

- The summary should condense the conversation to save tokens, but still contain enough information about the state of the plan to be actionable and apply the suggested changes.

Output only the summary of the current state of the plan and nothing else.
`

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
	return `Please summarize the text below using the 'summarize' function. Call 'summarize' with valid JSON that includes the 'summary' key. You must ALWAYS call the 'summarize' function in your reponse. Don't call any other function.
	
	Text:
					
	` + text
}

const PlanSummary = `
You are an AI summarizer that summarizes the conversation so far. The conversation so far is a plan to complete one or more programming tasks for a user. This conversation may begin with an existing summary of the plan.

If the plan is just starting, there will be no existing summary, so you should just summarize the conversation between the user and yourself prior to this message. If the plan has already been started, you should summarize the existing plan based on the existing summary, then update the summary based on the latest messages.

Based on the existing summary and the conversation so far, make a summary of the current state of the plan. 

- Begin with a summary of the user's messages, with particular focus any tasks they have given you. Your summary of the tasks should reflect the latest version of each task--if they have changed over time, summarize the latest state of each task that was given and omit anything that is now obsolete. Condense this information as much as possible while still being clear and retaining the meaning of the original messages.

- Next, if the plan has been broken down into subtasks, include ALL those subtasks *verbatim* in the summary as a numbered list, then mark whether each subtask has been implemented in code during the conversation. You must include ALL subtasks, even if they have not been mentioned in the latest messages. If this conversation began with an existing summary, include ALL the subtasks from the existing summary.

- Mark whether each subtask has been implemented in code during the conversation. A subtask has been implemented if *ANY* code blocks related to it have been included in the conversation. Do not be overly strict. Mark a subtask as implemented if it has been dealt with at all during the conversation. If a subtask has been implemented, mark it as implemented in code in the summary. 

- If a subtask has been further broken down into more subtasks, you should include those subtasks in the summary as a numbered list nested under the parent subtask. Just as with subtasks, you should include the name and a brief description of each sub-subtask and mark whether each sub-subtask has been implemented in code during the conversation.

- Any subtask or sub-subtask that has been marked as implemented in code in the existing summary *MUST* be marked as implemented in code in this new summary, even if they are not mentioned in the latest messages. It is EXTREMELY IMPORTANT that any subtask or sub-subtask that has been marked as implemented in the existing summary keeps the EXACT SAME status in this new summary.

- Each task, subtask, and sub-subtask should only appear once in the list. 

- A task, subtask, or sub-subtask can be removed from the list ONLY if it has not been implemented *and* the user has specifically asked for it not to be done, or it has been explicitly mentioned in the latest messages that it is no longer needed or relevant to the plan. If a subtask has been marked as implemented in code in the existing summary, it must not be removed from the list even if it is no longer relevant to the plan.

- Do NOT change the order of tasks, subtasks, or sub-subtasks in the list. If a task, subtask, or sub-subtask is listed as a subtask of another task, it should remain in that position in the list. If a subtask has sub-subtasks, those sub-subtasks should remain nested under that subtask.

- Last, summarize what has been done in the latest messages, and any next steps that have been described at the end of the latest message. Do NOT list the subtasks multiple times. Only list the subtasks once.

- Do not include code in the summary. Explain in words what has been done and what needs to be done.

- Treat the summary as *append-only*. Keep as much information as possible from the existing summary and add the new information from the latest messages. The summary is meant to be a record of the entire plan as it evolves over time. 

Output only the summary of the current state of the plan and nothing else.
`

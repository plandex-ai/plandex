package prompts

import (
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const SysExecStatusIsFinished = `You are an AI assistant that determines the execution status of a coding AI's plan for a programming task. Analyze the AI's latest message to determine whether the plan is finished. 

The plan *is finished* if all the plan's tasks and subtasks have been completed.

The plan *is finished* if no tasks or subtasks are mentioned or moved forward in the response. This can happen when the user doesn't have a task for the coding AI and is just asking a question or chatting.

The plan *is finished* if the response contains a sentence similar to "All tasks have been completed." near the end. 

If the response is a list of tasks, then the plan *is not finished*.

If the next task or subtask is mentioned at the end of the response, then the plan *is not finished*.

Return a JSON object with the 'finished' key set to true or false. You must ALWAYS call the 'planIsFinished' function. Don't call any other function.`

func GetExecStatusIsFinishedPrompt(message string) string {
	return SysExecStatusIsFinished + "\n\nLatest message from coding AI:\n" + message
}

var PlanIsFinishedFn = openai.FunctionDefinition{
	Name: "planIsFinished",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"finished": {
				Type:        jsonschema.Boolean,
				Description: "Whether the plan is finished.",
			},
		},
		Required: []string{"finished"},
	},
}

const SysExecStatusNeedsInput = `You are an AI assistant that determines the execution status of a coding AI's plan for a programming task. Analyze the AI's latest message to determine whether the plan needs more input. The plan needs more input if the coding AI requires the user to add more context, provide information, or answer questions the AI has asked.

When the coding AI needs more input, it will say something like "I need more information or context to make a plan for this task."

If the coding AI says or implies that additional information would be helpful or useful, but that information isn't *required* to continue the plan, then the plan *does not* need more input. It only needs more input if the AI says or implies that more information is necessary and required to continue. Return a JSON object with the 'needsInput' key set to true or false. You must ALWAYS call the 'planNeedsInput' function in your response. Don't call any other function.`

func GetExecStatusNeedsInputPrompt(message string) string {
	return SysExecStatusNeedsInput + "\nLatest message from coding AI:\n" + message
}

var PlanNeedsInputFn = openai.FunctionDefinition{
	Name: "planNeedsInput",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"needsInput": {
				Type:        jsonschema.Boolean,
				Description: "Whether the plan needs more input. If ambiguous or unclear, assume the plan does not need more input.",
			},
		},
		Required: []string{"needsInput"},
	},
}

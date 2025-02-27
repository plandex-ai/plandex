package prompts

import (
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const SysPlanNameXml = `You are an AI namer that creates a name for the plan. Most plans will be related to software development. You MUST output a valid XML response that includes a <planName> tag. The <planName> tag should contain a *short* lowercase file name for the plan content. Use dashes as word separators. No spaces, numbers, or special characters. **2-3 words max**. 1-2 words if you can. Shorten and abbreviate where possible. Do not use XML attributes - put all data as tag content.

Example response:
<planName>add-auth-system</planName>`

const SysPlanName = "You are an AI namer that creates a name for the plan. Most plans will be related to software development. Call the 'namePlan' function with a valid JSON object that includes the 'planName' key. 'planName' is a *short* lowercase file name for the plan content. Use dashes as word separators. No spaces, numbers, or special characters. **2-3 words max**. 1-2 words if you can. Shorten and abbreviate where possible. You must ALWAYS call the 'namePlan' function. Don't call any other function."

var PlanNameFn = openai.FunctionDefinition{
	Name: "namePlan",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"planName": {
				Type: jsonschema.String,
			},
		},
		Required: []string{"planName"},
	},
}

type PlanNameRes struct {
	PlanName string `json:"planName"`
}

func GetPlanNamePrompt(sysPrompt, text string) string {
	return sysPrompt + "\n\nContent:\n" + text
}

type PipedDataNameRes struct {
	Name string `json:"name"`
}

const SysPipedDataNameXml = `You are an AI namer that creates a name for output that has been piped into context. Take the output into account and also try to guess what command produced it if you can. You MUST output a valid XML response that includes a <name> tag. The <name> tag should contain a *short* lowercase name for the data. Use dashes as word separators. No spaces, numbers, or special characters. Shorten and abbreviate where possible. Do not use XML attributes - put all data as tag content.

Example response:
<name>git-status</name>`

const SysPipedDataName = "You are an AI namer that creates a name for output that has been piped into context. Take the output into account and also try to guess what command produced it if you can. Call the 'namePipedData' function with a valid JSON object that includes the 'name' key. 'name' is a *short* lowercase name for the data. Use dashes as word separators. No spaces, numbers, or special characters. Shorten and abbreviate where possible. You must ALWAYS call the 'namePipedData' function. Don't call any other function."

var PipedDataNameFn = openai.FunctionDefinition{
	Name: "namePipedData",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"name": {
				Type: jsonschema.String,
			},
		},
		Required: []string{"name"},
	},
}

func GetPipedDataNamePrompt(sysPrompt, text string) string {
	return SysPipedDataName + "\n\nContent:\n" + text
}

type NoteNameRes struct {
	Name string `json:"name"`
}

const SysNoteNameXml = `You are an AI namer that creates a name for an arbitrary text note. You MUST output a valid XML response that includes a <name> tag. The <name> tag should contain a *short* lowercase name for the data. Use dashes as word separators. No spaces, numbers, or special characters. Shorten and abbreviate where possible. Do not use XML attributes - put all data as tag content.

Example response:
<name>meeting-notes</name>`

const SysNoteName = "You are an AI namer that creates a name for an arbitrary text note. Call the 'nameNote' function with a valid JSON object that includes the 'name' key. 'name' is a *short* lowercase name for the data. Use dashes as word separators. No spaces, numbers, or special characters. Shorten and abbreviate where possible. You must ALWAYS call the 'nameNote' function. Don't call any other function."

var NoteNameFn = openai.FunctionDefinition{
	Name: "nameNote",
	Parameters: &jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"name": {
				Type: jsonschema.String,
			},
		},
		Required: []string{"name"},
	},
}

func GetNoteNamePrompt(sysPrompt, text string) string {
	return sysPrompt + "\n\nNote:\n" + text
}

dI want to update a number of LLM calls to potentially use XML instead of JSON-based tool calls based on their configuration in `AvailableModels` in `shared/ai_models.go`.

Here are the calls I want to update:

- commit message ('genPlanDescription' call)
- exec status ('execStatusShouldContinue' call) 
- naming functions - 'GenPlanName', 'GenPipedDataName', and 'GenNoteName'

Look at 'build_exec.go' for an example of how to extract XML from a response.

For each of the LLM calls:

- Before the call, you'll need to check the model config to see if the 'PreferredModelOutputFormat' field is set to xml/tool call json. We'll need to branch all the logic and prompts based on this.

- Add new prompting to output the same data using xml tags instead of a JSON function call if the model's 'PreferredModelOutputFormat' field is set to 'Xml'. do not use XML attributes, just simple tags. if there are multiple results in the json schema for the function call, update the prompt to output multiple tags. keep the rest of the prompts exactly the same. You can refactor shared aspects of the prompts between the xml version and the tool call json version.

- Look at the corresponding prompts for the build (in prompt/build.go) and use similar language for outputting xml tags in the xml versions of prompts.

- Update the post LLM call handling to extract the appropriate data using xml tags instead of json.

- Apart from the updated prompts, do not change other parameters in the LLM calls (like model, temperature, etc.)

- I don't want to have any nesting in the xml. the response should just contain multiple tags at the top level if multiple tags are needed. also, it must be clear in all cases that the output should be the content of the tag and not an attribute... brief examples must be included in every prompt as well for the xml versions.
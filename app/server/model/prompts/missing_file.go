package prompts

import "fmt"

func GetSkipMissingFilePrompt(path string) string {
	return fmt.Sprintf(`You *must not* generate content for the file %s. Skip this file and continue with the plan according to the 'Your instructions' section if there are any remaining tasks or subtasks. Don't repeat any part of the previous message. If there are no remaining tasks or subtasks, say "All tasks have been completed." per your instructions`, path)
}

const MissingFileContinueGeneratingPrompt = "Continue generating exactly where you left off in the previous message. Don't produce any other output before continuing or repeat any part of the previous message. Do *not* duplicate the last line of the previous response before continuing. Do *not* include triple backticks and a language name like '```python' or '```yaml' at the start of the response, since these have already been included in the previous message. Continue from where you left off seamlessly to generate the rest of the file block. You must include closing triple backticks at the end of the file block. When the file block is finished, continue with the plan according to the 'Your instructions' sections if there are any remaining tasks or subtasks. If there are no remaining tasks or subtasks, say 'All tasks have been completed.' per your instructions."

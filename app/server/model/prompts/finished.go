package prompts

const SysFinished = "You are an AI assistant that determines whether a software development plan has been finished or not. Analyze the conversation and decide whether all tasks have been completed."

func GetFinishedPrompt(conversation string) string {
	return SysFinished + "\n\nConversation:\n" + conversation
}

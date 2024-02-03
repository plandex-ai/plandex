package term

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

func OutputNoApiKeyMsg() {
	fmt.Fprintln(os.Stderr, color.New(color.Bold, color.FgHiRed).Sprintln("\n🚨 OPENAI_API_KEY environment variable is not set.")+color.New().Sprintln("\nSet it with:\n\nexport OPENAI_API_KEY=your-api-key\n\nThen try again.\n\n👉 If you don't have an OpenAI account, sign up here → https://platform.openai.com/signup\n\n🔑 Generate an api key here → https://platform.openai.com/api-keys"))
}

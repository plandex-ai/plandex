package term

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/plandex-ai/survey/v2"
)

func SelectFromList(msg string, options []string) (string, error) {
	var selected string
	prompt := &survey.Select{
		Message:       color.New(ColorHiMagenta, color.Bold).Sprint(msg),
		Options:       convertToStringSlice(options),
		FilterMessage: "",
	}
	err := survey.AskOne(prompt, &selected)
	if err != nil {
		if err.Error() == "interrupt" {
			os.Exit(0)
		}

		return "", err
	}

	return selected, nil
}

func convertToStringSlice[T any](input []T) []string {
	var result []string
	for _, v := range input {
		result = append(result, fmt.Sprint(v))
	}
	return result
}

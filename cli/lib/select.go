package lib

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/plandex-ai/survey/v2"
)

func SelectFromList(msg string, options []string) (string, error) {
	var selected string
	prompt := &survey.Select{
		Message:       color.New(color.FgHiMagenta, color.Bold).Sprint(msg),
		Options:       convertToStringSlice(options),
		FilterMessage: "",
	}
	err := survey.AskOne(prompt, &selected)
	if err != nil {
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

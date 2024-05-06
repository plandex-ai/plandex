package term

import (
	"fmt"
	"os"

	"github.com/cqroot/prompt"
	"github.com/cqroot/prompt/input"
	"github.com/eiannone/keyboard"
	"github.com/fatih/color"
)

func GetRequiredUserStringInput(msg string) (string, error) {
	res, err := GetUserStringInput(msg)
	if err != nil {
		return "", fmt.Errorf("failed to get user input: %s", err)
	}

	if res == "" {
		color.New(color.Bold, ColorHiRed).Println("🚨 This input is required")
		return GetRequiredUserStringInput(msg)
	}

	return res, nil
}

func GetUserStringInput(msg string) (string, error) {
	return GetUserStringInputWithDefault(msg, "")
}

func GetUserStringInputWithDefault(msg, def string) (string, error) {
	res, err := prompt.New().Ask(msg).Input(def)

	if err != nil && err.Error() == "user quit prompt" {
		os.Exit(0)
	}

	return res, err
}

func GetUserPasswordInput(msg string) (string, error) {
	res, err := prompt.New().Ask(msg).Input("", input.WithEchoMode(input.EchoPassword))

	if err != nil && err.Error() == "user quit prompt" {
		os.Exit(0)
	}

	return res, err
}

func GetUserKeyInput() (rune, error) {
	if err := keyboard.Open(); err != nil {
		return 0, fmt.Errorf("failed to open keyboard: %s", err)
	}
	defer func() {
		_ = keyboard.Close()
	}()

	char, _, err := keyboard.GetKey()
	if err != nil {
		return 0, fmt.Errorf("failed to read keypress: %s", err)
	}

	return char, nil
}

func ConfirmYesNo(fmtStr string, fmtArgs ...interface{}) (bool, error) {
	color.New(ColorHiMagenta, color.Bold).Printf(fmtStr+" (y)es | (n)o", fmtArgs...)
	color.New(ColorHiMagenta, color.Bold).Print("> ")

	char, err := GetUserKeyInput()
	if err != nil {
		return false, fmt.Errorf("failed to get user input: %s", err)
	}

	fmt.Println(string(char))
	if char == 'y' || char == 'Y' {
		return true, nil
	} else if char == 'n' || char == 'N' {
		return false, nil
	} else {
		fmt.Println()
		color.New(ColorHiRed, color.Bold).Print("Invalid input.\nEnter 'y' for yes or 'n' for no.\n\n")
		return ConfirmYesNo(fmtStr, fmtArgs...)
	}
}

func ConfirmYesNoCancel(fmtStr string, fmtArgs ...interface{}) (bool, bool, error) {
	color.New(ColorHiMagenta, color.Bold).Printf(fmtStr+" (y)es | (n)o | (c)ancel", fmtArgs...)
	color.New(ColorHiMagenta, color.Bold).Print("> ")

	char, err := GetUserKeyInput()
	if err != nil {
		return false, false, fmt.Errorf("failed to get user input: %s", err)
	}

	fmt.Println(string(char))
	if char == 'y' || char == 'Y' {
		return true, false, nil
	} else if char == 'n' || char == 'N' {
		return false, false, nil
	} else if char == 'c' || char == 'C' {
		return false, true, nil
	} else {
		fmt.Println()
		color.New(ColorHiRed, color.Bold).Print("Invalid input.\nEnter 'y' for yes, 'n' for no, or 'c' for cancel.\n\n")
		return ConfirmYesNoCancel(fmtStr, fmtArgs...)
	}
}

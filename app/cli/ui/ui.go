package ui

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"plandex/api"
	"plandex/term"

	"github.com/fatih/color"
	"github.com/pkg/browser"

	"github.com/plandex/plandex/shared"
)

func OpenAuthenticatedURL(msg, path string) {
	signInCode, apiErr := api.Client.CreateSignInCode()
	if apiErr != nil {
		log.Fatalf("Error creating sign in code: %v", apiErr)
	}

	host := api.GetApiHost()

	token := shared.UiSignInToken{
		Pin:        signInCode,
		RedirectTo: path,
	}

	jsonToken, err := json.Marshal(token)
	if err != nil {
		log.Fatalf("Error marshalling token: %v", err)
	}

	encodedToken := base64.URLEncoding.EncodeToString(jsonToken)

	url := fmt.Sprintf("%s/auth/%s", host, encodedToken)

	fmt.Printf(
		"%s\n\nIf it doesn't open automatically, use this URL:\n%s\n",
		color.New(term.ColorHiGreen).Sprintf(msg),
		url,
	)

	err = browser.OpenURL(url)
	if err != nil {
		fmt.Printf("Failed to open URL automatically: %v\n", err)
		fmt.Println("Please open the URL manually in your browser.")
	}
}

func OpenUnauthenticatedCloudURL(msg, path string) {
	host := api.CloudApiHost
	url := fmt.Sprintf("%s%s", host, path)

	fmt.Printf(
		"%s\n\nIf it doesn't open automatically, use this URL:\n%s\n",
		color.New(term.ColorHiGreen).Sprintf(msg),
		url,
	)

	err := browser.OpenURL(url)
	if err != nil {
		fmt.Printf("Failed to open URL automatically: %v\n", err)
		fmt.Println("Please open the URL manually in your browser.")
	}
}

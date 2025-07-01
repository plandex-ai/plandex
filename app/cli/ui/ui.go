package ui

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"plandex-cli/api"
	"plandex-cli/term"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/browser"

	shared "plandex-shared"
)

func OpenAuthenticatedURL(msg, path string) {
	signInCode, apiErr := api.Client.CreateSignInCode()
	if apiErr != nil {
		log.Fatalf("Error creating sign in code: %v", apiErr)
	}

	apiHost := api.GetApiHost()
	appHost := strings.Replace(apiHost, "api-v2.", "app.", 1)
	appHost = strings.Replace(appHost, "api.", "app.", 1)

	token := shared.UiSignInToken{
		Pin:        signInCode,
		RedirectTo: path,
	}

	jsonToken, err := json.Marshal(token)
	if err != nil {
		log.Fatalf("Error marshalling token: %v", err)
	}

	encodedToken := base64.URLEncoding.EncodeToString(jsonToken)

	url := fmt.Sprintf("%s/auth/%s", appHost, encodedToken)

	OpenURL(msg, url)
}

func OpenUnauthenticatedCloudURL(msg, path string) {
	apiHost := api.GetApiHost()
	appHost := strings.Replace(apiHost, "api-v2.", "app.", 1)

	url := fmt.Sprintf("%s%s", appHost, path)

	OpenURL(msg, url)
}

func OpenURL(msg, url string) {

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

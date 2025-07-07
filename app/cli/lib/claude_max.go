package lib

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"plandex-cli/term"
	"plandex-cli/types"
	"plandex-cli/ui"
	shared "plandex-shared"
	"strings"
	"time"

	"github.com/fatih/color"
)

const claudeMaxClientId = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
const claudeMaxScopes = "org:create_api_key user:profile user:inference"
const claudeMaxRedirect = "https://console.anthropic.com/oauth/code/callback"
const claudeMaxTokenUrl = "https://console.anthropic.com/v1/oauth/token"

func hasAnthropicModels(opts shared.ModelProviderOptions) bool {
	for _, opt := range opts {
		if opt.Config.Provider == shared.ModelProviderAnthropic {
			return true
		}
	}
	return false
}

func promptClaudeMaxIfNeeded() bool {
	orgUserConfig := MustGetOrgUserConfig()
	if orgUserConfig.PromptedClaudeMax {
		return false
	}

	term.StopSpinner()
	fmt.Println("ℹ️  The current model pack uses Anthropic models.\nIf you have a " + color.New(color.FgHiGreen, color.Bold).Sprint("Claude Pro or Max Subscription") + ", you can connect to it.\nPlandex will then use your Claude subscription for Anthropic model calls up to your limit.\n")

	res, err := term.ConfirmYesNo("Connect your Claude subscription?")
	if err != nil {
		term.OutputErrorAndExit("Error confirming claude connection: %v", err)
	}

	// update org user config to avoid prompting again
	orgUserConfig.PromptedClaudeMax = true
	MustUpdateOrgUserConfig(*orgUserConfig)

	if !res {
		fmt.Println()
		color.New(color.FgHiBlue).Println("To connect a Claude subscription later, run:\n" + term.ShowCmd("connect-claude"))
		fmt.Println()
		return false
	}

	ConnectClaudeMax()

	return true
}

func connectClaudeMaxIfNeeded() bool {
	accountCreds, err := GetAccountCredentials()
	if err != nil {
		term.OutputErrorAndExit("Error getting account credentials: %v", err)
	}

	if accountCreds == nil || accountCreds.ClaudeMax == nil {

		term.StopSpinner()
		fmt.Println("ℹ️  You connected a " + color.New(color.FgHiGreen, color.Bold).Sprint("Claude Pro or Max subscription,") + "\nbut credentials weren't found on this device.\n")

		res, err := term.ConfirmYesNo("Connect your Claude subscription?")
		if err != nil {
			term.OutputErrorAndExit("Error confirming claude connection: %v", err)
		}

		if res {
			ConnectClaudeMax()
			return true
		}
	}

	return false
}

func refreshClaudeMaxCredsIfNeeded() {
	accountCreds, err := GetAccountCredentials()
	if err != nil {
		term.OutputErrorAndExit("Error getting account credentials: %v", err)
	}
	if accountCreds == nil || accountCreds.ClaudeMax == nil {
		return
	}
	if !needsRefresh(accountCreds.ClaudeMax) || accountCreds.ClaudeMax.RefreshToken == "" {
		return
	}

	_, status, err := refreshCreds(accountCreds)
	if err != nil {
		if status == http.StatusUnauthorized {
			term.StopSpinner()
			color.New(color.FgHiYellow, color.Bold).Println("⚠️ Your Claude subscription's connection has been lost")
			fmt.Println()
			res, err := term.ConfirmYesNo("Reconnect your Claude subscription?")
			if err != nil {
				term.OutputErrorAndExit("Error confirming claude connection: %v", err)
			}
			if !res {
				accountCreds.ClaudeMax = nil
				if err := SetAccountCredentials(accountCreds); err != nil {
					term.OutputErrorAndExit("Error clearing Claude credentials: %v", err)
				}
				return
			}
			ConnectClaudeMax()
			return
		}
		term.OutputErrorAndExit("Error refreshing Claude credentials: %v", err)
	}
}

func ConnectClaudeMax() {
	connectClaudeMaxOauth()

	term.StartSpinner("")
	orgUserConfig := MustGetOrgUserConfig()
	orgUserConfig.UseClaudeSubscription = true
	MustUpdateOrgUserConfig(*orgUserConfig)

	term.StopSpinner()
	fmt.Println()
	fmt.Println("✅ Your Claude subscription is now connected")
	fmt.Println()

	color.New(color.FgHiBlue).Println("To disconnect, run:\n" + term.ShowCmd("disconnect-claude"))
	fmt.Println()
}

func DisconnectClaudeMax() {
	term.StartSpinner("")

	orgUserConfig := MustGetOrgUserConfig()
	orgUserConfig.UseClaudeSubscription = false
	MustUpdateOrgUserConfig(*orgUserConfig)

	accountCreds, err := GetAccountCredentials()
	if err != nil {
		term.OutputErrorAndExit("Error getting account credentials: %v", err)
	}

	if accountCreds != nil {
		accountCreds.ClaudeMax = nil
		if err := SetAccountCredentials(accountCreds); err != nil {
			term.OutputErrorAndExit("Error clearing Claude credentials: %v", err)
		}
	}

	term.StopSpinner()

	fmt.Println("✅ Your Claude subscription has been disconnected")
	fmt.Println()
	color.New(color.FgHiBlue).Println("To reconnect, run:\n" + term.ShowCmd("connect-claude"))
	fmt.Println()
}

func connectClaudeMaxOauth() {
	verifier, err := genCodeVerifier()
	if err != nil {
		term.OutputErrorAndExit("Error generating code verifier: %v", err)
	}
	challenge := sha256Base64(verifier)

	state, err := genCodeVerifier()
	if err != nil {
		term.OutputErrorAndExit("Error generating state: %v", err)
	}

	authURL := fmt.Sprintf(
		"https://claude.ai/oauth/authorize?code=true&client_id=%s&response_type=code&scope=%s&redirect_uri=%s&code_challenge=%s&code_challenge_method=S256&state=%s",
		claudeMaxClientId, url.QueryEscape(claudeMaxScopes), url.QueryEscape(claudeMaxRedirect), challenge, state,
	)

	term.StopSpinner()

	fmt.Println()
	ui.OpenURL("Opening Claude authentication page in your default browser...", authURL)
	fmt.Println()

	color.New(color.FgHiGreen, color.Bold).Println("📋 Click 'Authorize', copy the Authentication Code, then paste it below.\n")

	pastedCode, err := term.GetUserPasswordInput("Authentication Code:")
	if err != nil {
		term.OutputErrorAndExit("Error reading pasted authentication code: %v", err)
	}

	split := strings.SplitN(pastedCode, "#", 2)
	if len(split) != 2 {
		term.OutputErrorAndExit("Invalid authentication code: %s", pastedCode)
	}
	code := split[0]
	pastedState := split[1]

	if code == "" || pastedState != state {
		term.OutputErrorAndExit("Claude authentication failed: missing or mismatched oauth code/state")
	}
	term.StartSpinner("")

	tokens, err := exchangeCode(code, verifier, state)
	if err != nil {
		term.OutputErrorAndExit("Error exchanging code: %v", err)
	}

	creds := types.AccountCredentials{
		ClaudeMax: &types.OauthCreds{
			OauthResponse: *tokens,
			ExpiresAt:     time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second),
		},
	}
	if err := SetAccountCredentials(&creds); err != nil {
		term.OutputErrorAndExit("Error setting account credentials: %v", err)
	}
}

func exchangeCode(code, verifier, state string) (*types.OauthResponse, error) {
	body, _ := json.Marshal(map[string]any{
		"grant_type":    "authorization_code",
		"code":          code,
		"state":         state,
		"code_verifier": verifier,
		"redirect_uri":  claudeMaxRedirect,
		"client_id":     claudeMaxClientId,
	})
	req, err := http.NewRequest("POST", claudeMaxTokenUrl, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("token exchange failed - error creating request: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-beta", shared.AnthropicClaudeMaxBetaHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("token exchange failed - error reading body: %s", err)
		}
		return nil, fmt.Errorf("token exchange failed - status: %d, body: %s", resp.StatusCode, b)
	}
	var t types.OauthResponse
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return nil, err
	}
	return &t, nil
}

func genCodeVerifier() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func sha256Base64(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func needsRefresh(creds *types.OauthCreds) bool {
	// refresh an hour early so we can make multiple calls before it expires
	return time.Now().After(creds.ExpiresAt.Add(-1 * time.Hour))
}

func refreshCreds(accountCreds *types.AccountCredentials) (*types.OauthCreds, int, error) {
	creds := accountCreds.ClaudeMax
	if creds == nil {
		return nil, 0, fmt.Errorf("no stored Claude credentials")
	}

	body, err := json.Marshal(map[string]any{
		"grant_type":    "refresh_token",
		"refresh_token": creds.RefreshToken,
		"client_id":     claudeMaxClientId,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("refresh failed - marshal: %w", err)
	}

	req, err := http.NewRequest("POST", claudeMaxTokenUrl, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("refresh failed - create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-beta", shared.AnthropicClaudeMaxBetaHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("refresh failed - http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, 0, fmt.Errorf("refresh failed - read body: %w", err)
		}
		return nil, resp.StatusCode, fmt.Errorf("refresh failed - status %d: %s", resp.StatusCode, b)
	}

	var r types.OauthResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, 0, fmt.Errorf("refresh failed - decode: %w", err)
	}

	newCreds := &types.OauthCreds{
		OauthResponse: r,
		ExpiresAt:     time.Now().Add(time.Duration(r.ExpiresIn) * time.Second),
	}

	// persist updated creds
	accountCreds.ClaudeMax = newCreds
	if err := SetAccountCredentials(accountCreds); err != nil {
		return nil, 0, fmt.Errorf("refresh failed - save: %w", err)
	}

	return newCreds, resp.StatusCode, nil
}

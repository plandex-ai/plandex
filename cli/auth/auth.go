package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"plandex/fs"
	"plandex/term"
	"plandex/types"

	"github.com/plandex/plandex/shared"
)

var Current *types.ClientAuth
var apiClient types.ApiClient

func SetApiClient(client types.ApiClient) {
	apiClient = client
}

func MustResolveAuth() {
	if apiClient == nil {
		panic(fmt.Errorf("error resolving auth: api client not set"))
	}

	// load HomeAuthPath file into ClientAuth struct
	bytes, err := os.ReadFile(fs.HomeAuthPath)

	if err != nil {
		if os.IsNotExist(err) {
			err = promptAuth()

			if err != nil {
				panic(fmt.Errorf("error resolving auth: %v", err))
			}

			return
		} else {
			panic(fmt.Errorf("error reading auth.json: %v", err))
		}
	}

	var auth types.ClientAuth
	err = json.Unmarshal(bytes, &auth)
	if err != nil {
		panic(fmt.Errorf("error unmarshalling auth.json: %v", err))
	}

	Current = &auth
}

func LoadAccounts() ([]*types.ClientAccount, error) {
	bytes, err := os.ReadFile(fs.HomeAccountsPath)

	if err != nil {
		if os.IsNotExist(err) {
			// no accounts
			return []*types.ClientAccount{}, nil
		} else {
			return nil, fmt.Errorf("error reading accounts.json: %v", err)
		}
	}

	var accounts []*types.ClientAccount
	err = json.Unmarshal(bytes, &accounts)

	if err != nil {
		return nil, fmt.Errorf("error unmarshalling accounts.json: %v", err)
	}

	return accounts, nil
}

func StoreAccountIfNew(acct *types.ClientAccount) error {
	accounts, err := LoadAccounts()

	if err != nil {
		return fmt.Errorf("error loading accounts: %v", err)
	}

	for _, account := range accounts {
		if account.UserId == acct.UserId {
			return nil
		}
	}

	accounts = append(accounts, acct)

	bytes, err := json.Marshal(accounts)

	if err != nil {
		return fmt.Errorf("error marshalling accounts: %v", err)
	}

	err = os.WriteFile(fs.HomeAccountsPath, bytes, os.ModePerm)

	if err != nil {
		return fmt.Errorf("error writing accounts: %v", err)
	}

	return nil
}

func SetAuthHeader(req *http.Request) error {
	if Current == nil {
		return fmt.Errorf("error setting auth header: auth not loaded")
	}

	authHeader := shared.AuthHeader{
		Token: Current.Token,
		OrgId: Current.OrgId,
	}

	bytes, err := json.Marshal(authHeader)

	if err != nil {
		return fmt.Errorf("error marshalling auth header: %v", err)
	}

	// base64 encode
	token := base64.StdEncoding.EncodeToString(bytes)

	req.Header.Set("Authorization", "Bearer "+token)

	return nil
}

func writeCurrentAuth() error {
	if Current == nil {
		return fmt.Errorf("error writing auth: auth not loaded")
	}

	bytes, err := json.Marshal(Current)

	if err != nil {
		return fmt.Errorf("error marshalling auth: %v", err)
	}

	err = os.WriteFile(fs.HomeAuthPath, bytes, os.ModePerm)

	if err != nil {
		return fmt.Errorf("error writing auth: %v", err)
	}

	return nil
}

const (
	AuthFreeTrialOption = "Start a free trial on Plandex Cloud"
	AuthSignInOption    = "Sign in to an existing account on Plandex Cloud or another host"
	AuthCreateAccount   = "Create a new account on Plandex Cloud or another host"
)

func promptAuth() error {
	selected, err := term.SelectFromList("👋 Hey there! It looks like this is your first time using Plandex on this computer. What would you like to do?", []string{AuthFreeTrialOption, AuthSignInOption, AuthCreateAccount})

	if err != nil {
		return fmt.Errorf("error selecting auth option: %v", err)
	}

	switch selected {
	case AuthFreeTrialOption:
		err = startTrial()

		if err != nil {
			return fmt.Errorf("error starting trial: %v", err)
		}

	case AuthSignInOption:
		// sign in
		// TODO

	case AuthCreateAccount:
		// create account
		// TODO
	}

	return nil
}

func signIn() error {
	acccounts, err := LoadAccounts()

	if err != nil {
		return fmt.Errorf("error loading accounts: %v", err)
	}

	if len(acccounts) == 0 {
		err := promptSignInNewAccount()
		if err != nil {
			return fmt.Errorf("error signing in to new account: %v", err)
		}
	}

	var options []string
	for _, account := range acccounts {
		options = append(options, fmt.Sprintf("<%s> %s", account.UserName, account.Email))
	}

	options = append(options, "Sign in to another account")

	// either select from existing accounts or prompt for email

	return nil
}

const (
	SignInCloudOption = "Sign in to Plandex Cloud"
	SignInOtherOption = "Sign in to another host"
)

func promptSignInNewAccount() error {

	selected, err := term.SelectFromList("You can sign in to an existing account on Plandex Cloud or another host.", []string{SignInCloudOption, SignInOtherOption})

	if err != nil {
		return fmt.Errorf("error selecting sign in option: %v", err)
	}

	if selected == SignInCloudOption {
		email, err := term.GetUserStringInput("Your email:")

		if err != nil {
			return fmt.Errorf("error prompting email: %v", err)
		}
	} else {
		host, err := term.GetUserStringInput("Host:")

		if err != nil {
			return fmt.Errorf("error prompting host: %v", err)
		}

		email, err := term.GetUserStringInput("Your email:")

		if err != nil {
			return fmt.Errorf("error prompting email: %v", err)
		}

	}
}

func createAccount() error {

	return nil
}

func startTrial() error {
	term.StartSpinner("🌟 Starting trial...")

	res, err := apiClient.StartTrial()

	if err != nil {
		term.StopSpinner()
		return fmt.Errorf("error starting trial: %v", err)
	}

	term.StopSpinner()

	err = StoreAccountIfNew(&types.ClientAccount{
		Email:    res.Email,
		UserId:   res.UserId,
		UserName: res.UserName,
		Token:    res.Token,
		IsTrial:  true,
	})

	if err != nil {
		return fmt.Errorf("error storing trial account: %v", err)
	}

	Current = &types.ClientAuth{
		Email:    res.Email,
		UserId:   res.UserId,
		UserName: res.UserName,
		OrgId:    res.OrgId,
		OrgName:  res.OrgName,
		Token:    res.Token,
		IsTrial:  true,
	}

	err = writeCurrentAuth()

	if err != nil {
		return fmt.Errorf("error writing auth: %v", err)
	}

	return nil
}

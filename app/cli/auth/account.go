package auth

import (
	"fmt"
	"plandex-cli/term"

	shared "plandex-shared"

	"github.com/fatih/color"
)

const (
	AuthTrialOption   = "Start a trial on Plandex Cloud"
	AuthAccountOption = "Sign in, accept an invite, or create an account"
)

const AddAccountOption = "Add another account"

func SelectOrSignInOrCreate() error {
	accounts, err := loadAccounts()

	if err != nil {
		return fmt.Errorf("error loading accounts: %v", err)
	}

	if len(accounts) == 0 {
		err := promptSignInNewAccount()
		if err != nil {
			return fmt.Errorf("error signing in to new account: %v", err)
		}

		return nil
	}

	var options []string
	for _, account := range accounts {
		options = append(options, fmt.Sprintf("<%s> %s", account.UserName, account.Email))
	}

	options = append(options, AddAccountOption)

	// either select from existing accounts or sign in/create account

	selectedOpt, err := term.SelectFromList("Select an account:", options)

	if err != nil {
		return fmt.Errorf("error selecting account: %v", err)
	}

	if selectedOpt == AddAccountOption {
		err := promptSignInNewAccount()
		if err != nil {
			return fmt.Errorf("error prompting for sign in to new account: %v", err)
		}
		return nil
	}

	var selected *shared.ClientAccount
	for i, opt := range options {
		if selectedOpt == opt {
			selected = accounts[i]
			break
		}
	}

	if selected == nil {
		return fmt.Errorf("error selecting account: account not found")
	}

	selectedAuth := *selected

	setAuth(&shared.ClientAuth{
		ClientAccount: selectedAuth,
	})

	term.StartSpinner("")
	orgs, apiErr := apiClient.ListOrgs()
	term.StopSpinner()

	if apiErr != nil {
		return fmt.Errorf("error listing orgs: %v", apiErr.Msg)
	}

	org, err := resolveOrgAuth(orgs, selectedAuth.IsLocalMode)

	if err != nil {
		return fmt.Errorf("error resolving org: %v", err)
	}

	err = setAuth(&shared.ClientAuth{
		ClientAccount:        *selected,
		OrgId:                org.Id,
		OrgName:              org.Name,
		OrgIsTrial:           org.IsTrial,
		IntegratedModelsMode: org.IntegratedModelsMode,
	})

	if err != nil {
		return fmt.Errorf("error setting auth: %v", err)
	}

	_, apiErr = apiClient.GetOrgSession()

	if apiErr != nil {
		return fmt.Errorf("error getting org session: %v", apiErr.Msg)
	}

	fmt.Printf("✅ Signed in as %s | Org: %s\n", color.New(color.Bold, term.ColorHiGreen).Sprintf("<%s> %s", Current.UserName, Current.Email), color.New(term.ColorHiCyan).Sprint(Current.OrgName))
	fmt.Println()

	if !term.IsRepl {
		term.PrintCmds("", "")
	}

	return nil
}

func SignInWithCode(code, host string) error {
	term.StartSpinner("")
	res, apiErr := apiClient.SignIn(shared.SignInRequest{
		Pin:          code,
		IsSignInCode: true,
	}, host)
	term.StopSpinner()

	if apiErr != nil {
		return fmt.Errorf("error signing in: %v", apiErr.Msg)
	}

	return handleSignInResponse(res, host)
}

func promptInitialAuth() error {
	selected, err := term.SelectFromList("👋 Hey there!\nIt looks like this is your first time using Plandex on this computer.\nWhat would you like to do?", []string{AuthTrialOption, AuthAccountOption})

	if err != nil {
		return fmt.Errorf("error selecting auth option: %v", err)
	}

	switch selected {
	case AuthTrialOption:
		startTrial()

	case AuthAccountOption:
		err = SelectOrSignInOrCreate()

		if err != nil {
			return fmt.Errorf("error selecting or signing in to account: %v", err)
		}
	}

	return nil
}

const (
	SignInCloudOption = "Plandex Cloud"
	SignInLocalOption = "Local mode host"
	SignInOtherOption = "Another host"
)

func promptSignInNewAccount() error {
	selected, err := term.SelectFromList("Use Plandex Cloud or another host?", []string{SignInCloudOption, SignInLocalOption, SignInOtherOption})

	if err != nil {
		return fmt.Errorf("error selecting sign in option: %v", err)
	}

	var host string
	var email string

	if selected == SignInCloudOption {
		email, err = term.GetRequiredUserStringInput("Your email:")

		if err != nil {
			return fmt.Errorf("error prompting email: %v", err)
		}
	} else {
		if selected == SignInLocalOption {
			host, err = term.GetRequiredUserStringInputWithDefault("Host:", "http://localhost:8099")
		} else {
			host, err = term.GetRequiredUserStringInput("Host:")
		}

		if err != nil {
			return fmt.Errorf("error prompting host: %v", err)
		}

		if selected == SignInLocalOption {
			email = "local-admin@plandex.ai"
		} else {
			email, err = term.GetRequiredUserStringInput("Your email:")
		}

		if err != nil {
			return fmt.Errorf("error prompting email: %v", err)
		}
	}

	res, err := verifyEmail(email, host)

	if err != nil {
		return fmt.Errorf("error verifying email: %v", err)
	}

	if res.hasAccount {
		err := signIn(email, res.pin, host)
		if err != nil {
			return fmt.Errorf("error signing in: %v", err)
		}
	} else {
		err := createAccount(email, res.pin, host, res.isLocalMode)
		if err != nil {
			return fmt.Errorf("error creating account: %v", err)
		}
	}

	if !term.IsRepl {
		term.PrintCmds("", "")
	}

	return nil
}

type verifyEmailRes struct {
	hasAccount  bool
	isLocalMode bool
	pin         string
}

func verifyEmail(email, host string) (*verifyEmailRes, error) {
	term.StartSpinner("")
	res, apiErr := apiClient.CreateEmailVerification(email, host, "")
	term.StopSpinner()

	if apiErr != nil {
		return nil, fmt.Errorf("error creating email verification: %v", apiErr.Msg)
	}

	if res.IsLocalMode {
		return &verifyEmailRes{
			hasAccount:  res.HasAccount,
			isLocalMode: true,
			pin:         "",
		}, nil
	}

	fmt.Println("✉️  You'll now receive a 6 character pin by email. It will be valid for 5 minutes.")

	pin, err := term.GetUserPasswordInput("Please enter your pin:")

	if err != nil {
		return nil, fmt.Errorf("error prompting pin: %v", err)
	}

	return &verifyEmailRes{
		hasAccount:  res.HasAccount,
		isLocalMode: false,
		pin:         pin,
	}, nil
}

func signIn(email, pin, host string) error {
	term.StartSpinner("")
	res, apiErr := apiClient.SignIn(shared.SignInRequest{
		Email: email,
		Pin:   pin,
	}, host)
	term.StopSpinner()

	if apiErr != nil {
		return fmt.Errorf("error signing in: %v", apiErr.Msg)
	}

	return handleSignInResponse(res, host)
}

func handleSignInResponse(res *shared.SessionResponse, host string) error {
	isLocalMode := host != "" && res.IsLocalMode

	err := setAuth(&shared.ClientAuth{
		ClientAccount: shared.ClientAccount{
			Email:       res.Email,
			UserId:      res.UserId,
			UserName:    res.UserName,
			Token:       res.Token,
			IsTrial:     false,
			IsCloud:     host == "",
			Host:        host,
			IsLocalMode: isLocalMode,
		},
	})

	if err != nil {
		return fmt.Errorf("error setting auth: %v", err)
	}

	org, err := resolveOrgAuth(res.Orgs, isLocalMode)

	if err != nil {
		return fmt.Errorf("error resolving org: %v", err)
	}

	Current.OrgId = org.Id
	Current.OrgName = org.Name
	Current.IntegratedModelsMode = org.IntegratedModelsMode

	err = writeCurrentAuth()

	if err != nil {
		return fmt.Errorf("error writing auth: %v", err)
	}

	fmt.Printf("✅ Signed in as %s | Org: %s\n", color.New(color.Bold, term.ColorHiGreen).Sprintf("<%s> %s", Current.UserName, Current.Email), color.New(term.ColorHiCyan).Sprint(Current.OrgName))
	fmt.Println()

	return nil
}

func createAccount(email, pin, host string, isLocalMode bool) error {
	var name string

	if isLocalMode {
		name = "Local Admin"
	} else {
		var err error
		name, err = term.GetUserStringInput("Your name:")

		if err != nil {
			return fmt.Errorf("error prompting name: %v", err)
		}
	}

	term.StartSpinner("🌟 Creating account...")
	res, apiErr := apiClient.CreateAccount(shared.CreateAccountRequest{
		Email:    email,
		UserName: name,
		Pin:      pin,
	}, host)
	term.StopSpinner()

	if apiErr != nil {
		return fmt.Errorf("error creating account: %v", apiErr.Msg)
	}

	if res.IsLocalMode {
		isLocalMode = true
	}

	err := setAuth(&shared.ClientAuth{
		ClientAccount: shared.ClientAccount{
			Email:       res.Email,
			UserId:      res.UserId,
			UserName:    res.UserName,
			Token:       res.Token,
			IsTrial:     false,
			IsCloud:     host == "",
			Host:        host,
			IsLocalMode: isLocalMode,
		},
	})

	if err != nil {
		return fmt.Errorf("error setting auth: %v", err)
	}

	org, err := resolveOrgAuth(res.Orgs, isLocalMode)

	if err != nil {
		return fmt.Errorf("error resolving org: %v", err)
	}

	if org == nil {
		return fmt.Errorf("no org selected")
	}

	Current.OrgId = org.Id
	Current.OrgName = org.Name
	Current.IntegratedModelsMode = org.IntegratedModelsMode

	err = writeCurrentAuth()

	if err != nil {
		return fmt.Errorf("error writing auth: %v", err)
	}

	fmt.Printf("✅ Signed in as %s | Org: %s\n", color.New(color.Bold, term.ColorHiGreen).Sprintf("<%s> %s", Current.UserName, Current.Email), color.New(term.ColorHiCyan).Sprint(Current.OrgName))
	fmt.Println()

	return nil
}

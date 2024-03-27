package auth

import (
	"fmt"
	"plandex/term"
	"plandex/types"

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
)

const (
	AuthFreeTrialOption = "Start an anonymous trial on Plandex Cloud (no email required)"
	AuthAccountOption   = "Sign in, accept an invite, or create an account"
)

func promptInitialAuth() error {
	selected, err := term.SelectFromList("👋 Hey there!\nIt looks like this is your first time using Plandex on this computer.\nWhat would you like to do?", []string{AuthFreeTrialOption, AuthAccountOption})

	if err != nil {
		return fmt.Errorf("error selecting auth option: %v", err)
	}

	switch selected {
	case AuthFreeTrialOption:
		err = startTrial()

		if err != nil {
			return fmt.Errorf("error starting trial: %v", err)
		}

	case AuthAccountOption:
		err = SelectOrSignInOrCreate()

		if err != nil {
			return fmt.Errorf("error selecting or signing in to account: %v", err)
		}
	}

	return nil
}

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

	var selected *types.ClientAccount
	for i, opt := range options {
		if selectedOpt == opt {
			selected = accounts[i]
			break
		}
	}

	if selected == nil {
		return fmt.Errorf("error selecting account: account not found")
	}

	setAuth(&types.ClientAuth{
		ClientAccount: *selected,
	})

	term.StartSpinner("")
	orgs, apiErr := apiClient.ListOrgs()
	term.StopSpinner()

	if apiErr != nil {
		return fmt.Errorf("error listing orgs: %v", apiErr.Msg)
	}

	orgId, orgName, err := resolveOrgAuth(orgs)

	if err != nil {
		return fmt.Errorf("error resolving org: %v", err)
	}

	err = setAuth(&types.ClientAuth{
		ClientAccount: *selected,
		OrgId:         orgId,
		OrgName:       orgName,
	})

	if err != nil {
		return fmt.Errorf("error setting auth: %v", err)
	}

	apiErr = apiClient.GetOrgSession()

	if apiErr != nil {
		return fmt.Errorf("error getting org session: %v", apiErr.Msg)
	}

	fmt.Printf("✅ Signed in as %s | Org: %s\n", color.New(color.Bold, term.ColorHiGreen).Sprintf("<%s> %s", Current.UserName, Current.Email), color.New(term.ColorHiCyan).Sprint(Current.OrgName))
	fmt.Println()

	term.PrintCmds("", "new", "plans")

	return nil
}

const (
	SignInCloudOption = "Plandex Cloud"
	SignInOtherOption = "Another host"
)

func promptSignInNewAccount() error {
	selected, err := term.SelectFromList("Use Plandex Cloud or another host?", []string{SignInCloudOption, SignInOtherOption})

	if err != nil {
		return fmt.Errorf("error selecting sign in option: %v", err)
	}

	var host string
	var email string

	if selected == SignInCloudOption {
		email, err = term.GetUserStringInput("Your email:")

		if err != nil {
			return fmt.Errorf("error prompting email: %v", err)
		}
	} else {
		host, err = term.GetUserStringInput("Host:")

		if err != nil {
			return fmt.Errorf("error prompting host: %v", err)
		}

		email, err = term.GetUserStringInput("Your email:")

		if err != nil {
			return fmt.Errorf("error prompting email: %v", err)
		}
	}

	hasAccount, pin, err := verifyEmail(email, host)

	if err != nil {
		return fmt.Errorf("error verifying email: %v", err)
	}

	if hasAccount {
		return signIn(email, pin, host)
	} else {
		return createAccount(email, pin, host)
	}
}

func verifyEmail(email, host string) (bool, string, error) {

	term.StartSpinner("")
	res, apiErr := apiClient.CreateEmailVerification(email, host, "")
	term.StopSpinner()

	if apiErr != nil {
		return false, "", fmt.Errorf("error creating email verification: %v", apiErr.Msg)
	}

	fmt.Println("✉️  You'll now receive a 6 character pin by email. It will be valid for 5 minutes.")

	pin, err := term.GetUserPasswordInput("Please enter your pin:")

	if err != nil {
		return false, "", fmt.Errorf("error prompting pin: %v", err)
	}

	return res.HasAccount, pin, nil
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

	orgId, orgName, err := resolveOrgAuth(res.Orgs)

	if err != nil {
		return fmt.Errorf("error resolving org: %v", err)
	}

	err = setAuth(&types.ClientAuth{
		ClientAccount: types.ClientAccount{
			Email:    res.Email,
			UserId:   res.UserId,
			UserName: res.UserName,
			Token:    res.Token,
			IsTrial:  false,
			IsCloud:  host == "",
		},
		OrgId:   orgId,
		OrgName: orgName,
	})

	if err != nil {
		return fmt.Errorf("error setting auth: %v", err)
	}

	return nil
}

func createAccount(email, pin, host string) error {

	name, err := term.GetUserStringInput("Your name:")

	if err != nil {
		return fmt.Errorf("error prompting name: %v", err)
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

	err = setAuth(&types.ClientAuth{
		ClientAccount: types.ClientAccount{
			Email:    res.Email,
			UserId:   res.UserId,
			UserName: res.UserName,
			Token:    res.Token,
			IsTrial:  false,
			IsCloud:  host == "",
		},
	})

	if err != nil {
		return fmt.Errorf("error setting auth: %v", err)
	}

	orgId, orgName, err := resolveOrgAuth(res.Orgs)

	if err != nil {
		return fmt.Errorf("error resolving org: %v", err)
	}

	Current.OrgId = orgId
	Current.OrgName = orgName

	err = writeCurrentAuth()

	if err != nil {
		return fmt.Errorf("error writing auth: %v", err)
	}

	return nil
}

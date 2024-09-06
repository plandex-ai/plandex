package auth

import (
	"fmt"
	"plandex/term"

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
)

func ConvertTrial() error {
	email, err := term.GetRequiredUserStringInput("Your email:")

	if err != nil {
		return fmt.Errorf("error prompting email: %v", err)
	}

	hasAccount, pin, err := verifyEmail(email, "")

	if err != nil {
		return fmt.Errorf("error verifying email: %v", err)
	}

	if hasAccount {
		term.OutputErrorAndExit("Can't convert a trial into an account that already exists")
	}

	name, err := term.GetUserStringInput("Your name:")

	if err != nil {
		return fmt.Errorf("error prompting name: %v", err)
	}

	orgName, err := term.GetRequiredUserStringInput("Org name:")

	if err != nil {
		return fmt.Errorf("error prompting org name: %v", err)
	}

	autoAddDomainUsers, err := promptAutoAddUsersIfValid(email)

	if err != nil {
		return fmt.Errorf("error prompting auto add domain users: %v", err)
	}

	term.StartSpinner("")
	res, apiErr := apiClient.ConvertTrial(shared.ConvertTrialRequest{
		Email:                 email,
		Pin:                   pin,
		UserName:              name,
		OrgName:               orgName,
		OrgAutoAddDomainUsers: autoAddDomainUsers,
	})
	term.StopSpinner()

	if apiErr != nil {
		return fmt.Errorf("error converting trial: %v", apiErr)
	}

	err = setAuth(&shared.ClientAuth{
		ClientAccount: shared.ClientAccount{
			Email:    res.Email,
			UserId:   res.UserId,
			UserName: res.UserName,
			Token:    res.Token,
			IsCloud:  true,
			IsTrial:  false,
		},
		OrgId:                res.Orgs[0].Id,
		OrgName:              res.Orgs[0].Id,
		OrgIsTrial:           res.Orgs[0].IsTrial,
		IntegratedModelsMode: res.Orgs[0].IntegratedModelsMode,
	})

	if err != nil {
		return fmt.Errorf("error setting auth: %v", err)
	}

	return nil
}

const (
	TrialIntegratedModelsOption = "Integrated Models Mode\n   • no OpenAI (or other provider) account required\n   • Use OpenAI, Anthropic, and open source models\n   • start with $5 trial"
	TrialBYOModelsOption        = "BYO API Key Mode\n   • requires OpenAI account (or another provider) and API key\n   • start with free trial"
)

func startTrial() error {
	fmt.Printf("\n☁️  There are %s to use Plandex Cloud.\n\nYou can use %s to purchase LLM credits from Plandex directly, avoiding rate limits and the need to manage model provider accounts and API keys.\n\nOr you can use %s to use Plandex with your own account and API key from OpenAI or another model provider.\n\n",
		color.New(color.Bold).Sprint("two ways"),
		color.New(color.Bold, term.ColorHiYellow).Sprint("Integrated Models Mode"),
		color.New(color.Bold, term.ColorHiYellow).Sprint("BYO API Key Mode"),
	)

	selected, err := term.SelectFromList("Which do you want to start with?", []string{TrialIntegratedModelsOption, TrialBYOModelsOption})

	if err != nil {
		return fmt.Errorf("error selecting integrated models option: %v", err)
	}

	integratedModelsMode := selected == TrialIntegratedModelsOption

	email, err := term.GetRequiredUserStringInput("Your email:")

	if err != nil {
		return fmt.Errorf("error prompting email: %v", err)
	}

	hasAccount, pin, err := verifyEmail(email, "")

	if err != nil {
		return fmt.Errorf("error verifying email: %v", err)
	}

	if hasAccount {
		term.OutputErrorAndExit("Can't convert a trial into an account that already exists")
	}

	name, err := term.GetUserStringInput("Your name:")

	if err != nil {
		return fmt.Errorf("error prompting name: %v", err)
	}

	orgName, err := term.GetRequiredUserStringInput("Org name:")

	if err != nil {
		return fmt.Errorf("error prompting org name: %v", err)
	}

	autoAddDomainUsers, err := promptAutoAddUsersIfValid(email)

	if err != nil {
		return fmt.Errorf("error prompting auto add domain users: %v", err)
	}

	term.StartSpinner("🌟 Starting trial...")
	res, apiErr := apiClient.StartTrial(shared.StartTrialRequest{
		Account: shared.CreateAccountRequest{
			Email:    email,
			Pin:      pin,
			UserName: name,
		},
		Org: shared.CreateCloudOrgRequest{
			CreateOrgRequest: shared.CreateOrgRequest{
				Name:               orgName,
				AutoAddDomainUsers: autoAddDomainUsers,
			},
			IntegratedModelsMode: integratedModelsMode,
		},
	})
	term.StopSpinner()

	if apiErr != nil {
		return fmt.Errorf("error starting trial: %v", apiErr)
	}

	err = setAuth(&shared.ClientAuth{
		ClientAccount: shared.ClientAccount{
			Email:    email,
			UserId:   res.UserId,
			UserName: name,
			Token:    res.Token,
			IsCloud:  true,
			IsTrial:  true,
		},
		OrgId:                res.OrgId,
		OrgName:              orgName,
		IntegratedModelsMode: integratedModelsMode,
	})

	if err != nil {
		return fmt.Errorf("error setting auth: %v", err)
	}

	return nil
}

// func startTrial() error {

// 	term.StartSpinner("🌟 Starting trial...")

// 	res, apiErr := apiClient.StartTrial()

// 	term.StopSpinner()
// 	if apiErr != nil {
// 		return fmt.Errorf("error starting trial: %v", apiErr.Msg)
// 	}

// 	err := setAuth(&shared.ClientAuth{
// 		ClientAccount: shared.ClientAccount{
// 			Email:    res.Email,
// 			UserId:   res.UserId,
// 			UserName: res.UserName,
// 			Token:    res.Token,
// 			IsTrial:  true,
// 			IsCloud:  true,
// 		},
// 		OrgId:   res.OrgId,
// 		OrgName: res.OrgName,
// 	})

// 	if err != nil {
// 		return fmt.Errorf("error setting auth: %v", err)
// 	}

// 	return nil
// }

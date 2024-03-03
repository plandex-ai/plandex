package auth

import (
	"fmt"
	"plandex/term"
	"plandex/types"

	"github.com/plandex/plandex/shared"
)

func ConvertTrial() error {
	email, err := term.GetUserStringInput("Your email:")

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

	orgName, err := term.GetUserStringInput("Org name:")

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

	err = setAuth(&types.ClientAuth{
		ClientAccount: types.ClientAccount{
			Email:    res.Email,
			UserId:   res.UserId,
			UserName: res.UserName,
			Token:    res.Token,
			IsCloud:  true,
			IsTrial:  false,
		},
		OrgId:   res.Orgs[0].Id,
		OrgName: res.Orgs[0].Id,
	})

	if err != nil {
		return fmt.Errorf("error setting auth: %v", err)
	}

	return nil
}

func startTrial() error {
	term.StartSpinner("🌟 Starting trial...")

	res, apiErr := apiClient.StartTrial()

	term.StopSpinner()
	if apiErr != nil {
		return fmt.Errorf("error starting trial: %v", apiErr.Msg)
	}

	err := setAuth(&types.ClientAuth{
		ClientAccount: types.ClientAccount{
			Email:    res.Email,
			UserId:   res.UserId,
			UserName: res.UserName,
			Token:    res.Token,
			IsTrial:  true,
			IsCloud:  true,
		},
		OrgId:   res.OrgId,
		OrgName: res.OrgName,
	})

	if err != nil {
		return fmt.Errorf("error setting auth: %v", err)
	}

	return nil
}

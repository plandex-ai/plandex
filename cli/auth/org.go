package auth

import (
	"fmt"
	"plandex/term"
	"plandex/types"
	"strings"

	"github.com/plandex/plandex/shared"
)

func resolveOrgAuth(orgs []*shared.Org) (string, string, error) {
	var org *shared.Org
	var err error

	if len(orgs) == 0 {
		org, err = promptNoOrgs()

		if err != nil {
			return "", "", fmt.Errorf("error prompting no orgs: %v", err)
		}

	} else if len(orgs) == 1 {
		org = orgs[0]
	} else {
		org, err = selectOrg(orgs)

		if err != nil {
			return "", "", fmt.Errorf("error selecting org: %v", err)
		}
	}

	var (
		orgId   string
		orgName string
	)

	if org != nil {
		orgId = org.Id
		orgName = org.Name
	}

	return orgId, orgName, nil
}

func createAccount(email, pin, host string) error {

	name, err := term.GetUserStringInput("Your name:")

	if err != nil {
		return fmt.Errorf("error prompting name: %v", err)
	}

	res, apiErr := apiClient.CreateAccount(shared.CreateAccountRequest{
		Email:    email,
		UserName: name,
		Pin:      pin,
	}, host)

	if apiErr != nil {
		return fmt.Errorf("error creating account: %v", apiErr.Msg)
	}

	err = setAuth(&types.ClientAuth{
		ClientAccount: types.ClientAccount{
			Email:    res.Email,
			UserId:   res.UserId,
			UserName: res.UserName,
			Token:    res.Token,
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

func promptNoOrgs() (*shared.Org, error) {
	fmt.Println("🧐 You don't have access to any orgs yet.\n\nTo join an existing org, ask an admin to either invite you directly or give your whole email domain access.\n\nOtherwise, you can go ahead and create a new org.")

	shouldCreate, err := term.ConfirmYesNo("Create a new org now?")

	if err != nil {
		return nil, fmt.Errorf("error prompting create org: %v", err)
	}

	if shouldCreate {
		return createOrg()
	}

	return nil, nil
}

func createOrg() (*shared.Org, error) {
	name, err := term.GetUserStringInput("Org name:")
	if err != nil {
		return nil, fmt.Errorf("error prompting org name: %v", err)
	}

	autoAddDomainUsers, err := promptAutoAddUsersIfValid(Current.Email)

	if err != nil {
		return nil, fmt.Errorf("error prompting auto add domain users: %v", err)
	}

	res, apiErr := apiClient.CreateOrg(shared.CreateOrgRequest{
		Name:               name,
		AutoAddDomainUsers: autoAddDomainUsers,
	})

	if apiErr != nil {
		return nil, fmt.Errorf("error creating org: %v", apiErr.Msg)
	}

	return &shared.Org{Id: res.Id, Name: name}, nil
}

func promptAutoAddUsersIfValid(email string) (bool, error) {
	userDomain := strings.Split(Current.Email, "@")[1]
	var autoAddDomainUsers bool
	var err error
	if !shared.IsEmailServiceDomain(userDomain) {
		autoAddDomainUsers, err = term.ConfirmYesNo(fmt.Sprintf("Do you want to allow any user with an email ending in @%s to auto-join this org?", userDomain))

		if err != nil {
			return false, err
		}
	}
	return autoAddDomainUsers, nil
}

const CreateOrgOption = "Create a new org"

func selectOrg(orgs []*shared.Org) (*shared.Org, error) {
	var options []string
	for _, org := range orgs {
		options = append(options, org.Name)
	}
	options = append(options, CreateOrgOption)

	selected, err := term.SelectFromList("Select an org:", options)

	if err != nil {
		return nil, fmt.Errorf("error selecting org: %v", err)
	}

	if selected == CreateOrgOption {
		return createOrg()
	}

	var selectedOrg *shared.Org
	for _, org := range orgs {
		if org.Name == selected {
			selectedOrg = org
			break
		}
	}

	if selectedOrg == nil {
		return nil, fmt.Errorf("error selecting org: org not found")
	}

	return selectedOrg, nil
}

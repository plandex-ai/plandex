package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"plandex-cli/fs"

	shared "plandex-shared"
)

var Current *shared.ClientAuth

func loadAccounts() ([]*shared.ClientAccount, error) {
	bytes, err := os.ReadFile(fs.HomeAccountsPath)

	if err != nil {
		if os.IsNotExist(err) {
			// no accounts
			return []*shared.ClientAccount{}, nil
		} else {
			return nil, fmt.Errorf("error reading accounts.json: %v", err)
		}
	}

	var accounts []*shared.ClientAccount
	err = json.Unmarshal(bytes, &accounts)

	if err != nil {
		return nil, fmt.Errorf("error unmarshalling accounts.json: %v", err)
	}

	return accounts, nil
}

func setAuth(auth *shared.ClientAuth) error {
	err := storeAccount(&auth.ClientAccount)

	if err != nil {
		return fmt.Errorf("error storing account: %v", err)
	}

	Current = auth

	err = writeCurrentAuth()

	if err != nil {
		return fmt.Errorf("error writing auth: %v", err)
	}

	return nil
}

func storeAccount(toStore *shared.ClientAccount) error {
	accounts, err := loadAccounts()

	if err != nil {
		return fmt.Errorf("error loading accounts: %v", err)
	}

	found := false
	for i, account := range accounts {
		if account.UserId == toStore.UserId {
			accounts[i] = toStore
			found = true
			break
		}
	}

	if !found {
		accounts = append(accounts, toStore)
	}

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

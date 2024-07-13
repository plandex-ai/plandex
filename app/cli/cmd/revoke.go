package cmd

import (
	"fmt"
	"plandex/api"
	"plandex/auth"
	"plandex/term"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

var revokeCmd = &cobra.Command{
	Use:   "revoke [email]",
	Short: "Revoke an invite or remove a user from the org",
	Run:   revoke,
	Args:  cobra.MaximumNArgs(1),
}

func init() {
	RootCmd.AddCommand(revokeCmd)
}

func revoke(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	email := ""
	if len(args) > 0 {
		email = args[0]
	}

	var userResp *shared.ListUsersResponse
	var pendingInvites []*shared.Invite
	errCh := make(chan error)

	term.StartSpinner("")

	go func() {
		var err *shared.ApiError
		userResp, err = api.Client.ListUsers()
		if err != nil {
			errCh <- fmt.Errorf("error fetching users: %s", err.Msg)
			return
		}
		errCh <- nil
	}()

	go func() {
		var err *shared.ApiError
		pendingInvites, err = api.Client.ListPendingInvites()
		if err != nil {
			errCh <- fmt.Errorf("error fetching pending invites: %s", err.Msg)
			return
		}
		errCh <- nil
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil {
			term.StopSpinner()
			term.OutputErrorAndExit(err.Error())
		}
	}

	term.StopSpinner()

	type userInfo struct {
		Id       string
		IsInvite bool
	}

	emailToUserMap := make(map[string]userInfo)
	labelToEmail := make(map[string]string)

	// Combine users and invites for selection
	combinedList := make([]string, 0, len(userResp.Users)+len(pendingInvites))
	for _, user := range userResp.Users {
		label := fmt.Sprintf("%s <%s>", user.Name, user.Email)
		labelToEmail[label] = user.Email
		combinedList = append(combinedList, label)
		emailToUserMap[user.Email] = userInfo{Id: user.Id, IsInvite: false}
	}
	for _, invite := range pendingInvites {
		label := fmt.Sprintf("%s <%s> (invite pending)", invite.Name, invite.Email)
		labelToEmail[label] = invite.Email
		combinedList = append(combinedList, label)
		emailToUserMap[invite.Email] = userInfo{Id: invite.Id, IsInvite: true}
	}

	if email == "" {
		selected, err := term.SelectFromList("Select a user or invite:", combinedList)
		if err != nil {
			term.OutputErrorAndExit("Error selecting item to revoke: %v", err)
		}

		email = labelToEmail[selected]
	}

	if email == "" {
		term.OutputErrorAndExit("No user or invite selected")
	}

	// Determine if email belongs to a user or an invite and revoke accordingly
	if userInfo, exists := emailToUserMap[email]; exists {
		if userInfo.IsInvite {
			if err := api.Client.DeleteInvite(userInfo.Id); err != nil {
				term.OutputErrorAndExit("Failed to revoke invite: %v", err)
			}
			fmt.Println("✅ Invite revoked")
		} else {
			if err := api.Client.DeleteUser(userInfo.Id); err != nil {
				term.OutputErrorAndExit("Failed to remove user: %v", err)
			}
			fmt.Println("✅ User removed")
		}
	} else {
		term.OutputErrorAndExit("No user or pending invite found for email '%s'", email)
	}
}

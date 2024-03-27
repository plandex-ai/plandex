package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/term"

	"github.com/olekukonko/tablewriter"
	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "List all users and pending invites and the current org",
	Run:   listUsersAndInvites,
}

func init() {
	RootCmd.AddCommand(usersCmd)
}

func listUsersAndInvites(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	var userResp *shared.ListUsersResponse
	var pendingInvites []*shared.Invite
	var orgRoles []*shared.OrgRole

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

	go func() {
		var err *shared.ApiError
		orgRoles, err = api.Client.ListOrgRoles()
		if err != nil {
			errCh <- fmt.Errorf("error fetching org roles: %s", err.Msg)
			return
		}
		errCh <- nil

	}()

	for i := 0; i < 3; i++ {
		err := <-errCh
		if err != nil {
			term.StopSpinner()
			term.OutputErrorAndExit("%v", err)
		}
	}

	term.StopSpinner()

	orgRolesById := make(map[string]*shared.OrgRole)
	for _, role := range orgRoles {
		orgRolesById[role.Id] = role
	}

	// Display users and pending invites in a table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Email", "Name", "Role", "Status"})

	for _, user := range userResp.Users {
		table.Append([]string{user.Email, user.Name, orgRolesById[userResp.OrgUsersByUserId[user.Id].OrgRoleId].Label, "Active"})
	}

	for _, invite := range pendingInvites {
		table.Append([]string{invite.Email, invite.Name, orgRolesById[invite.OrgRoleId].Label, "Pending"})
	}

	table.Render()
}

package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

func CreateInvite(invite *Invite, tx *sql.Tx) error {
	err := tx.QueryRow("INSERT INTO invites (org_id, email, name, inviter_id, org_role_id) VALUES ($1, $2, $3, $4, $5) RETURNING id", invite.OrgId, invite.Email, invite.Name, invite.InviterId, invite.OrgRoleId).Scan(&invite.Id)

	if err != nil {
		return fmt.Errorf("error creating invite: %v", err)
	}

	return nil
}

func GetInvite(id string) (*Invite, error) {
	var invite Invite
	err := Conn.Get(&invite, "SELECT * FROM invites WHERE id = $1", id)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("error getting invite: %v", err)
	}

	return &invite, nil
}

func GetActiveInviteByEmail(orgId, email string) (*Invite, error) {
	var invite Invite
	err := Conn.Get(&invite, "SELECT * FROM invites WHERE org_id = $1 AND email = $2 AND accepted_at IS NULL", orgId, email)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("error getting invite: %v", err)
	}

	return &invite, nil
}

func ListPendingInvites(orgId string) ([]*Invite, error) {
	var invites []*Invite
	err := Conn.Select(&invites, "SELECT * FROM invites WHERE org_id = $1 AND accepted_at IS NULL", orgId)

	if err != nil {
		return nil, fmt.Errorf("error getting pending invites for org: %v", err)
	}

	return invites, nil
}

func ListAllInvites(orgId string) ([]*Invite, error) {
	var invites []*Invite
	err := Conn.Select(&invites, "SELECT * FROM invites WHERE org_id = $1", orgId)

	if err != nil {
		return nil, fmt.Errorf("error getting all invites for org: %v", err)
	}

	return invites, nil
}

func ListAcceptedInvites(orgId string) ([]*Invite, error) {
	var invites []*Invite
	err := Conn.Select(&invites, "SELECT * FROM invites WHERE org_id = $1 AND accepted_at IS NOT NULL", orgId)

	if err != nil {
		return nil, fmt.Errorf("error getting accepted invites for org: %v", err)
	}

	return invites, nil
}

func GetPendingInvitesForEmail(email string) ([]*Invite, error) {
	email = strings.ToLower(email)
	var invites []*Invite
	err := Conn.Select(&invites, "SELECT * FROM invites WHERE email = $1 AND accepted_at IS NULL", email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("error getting invites and org names for email: %v", err)
	}

	return invites, nil
}

func DeleteInvite(id string, tx *sql.Tx) error {
	query := "DELETE FROM invites WHERE id = $1"
	var err error

	if tx == nil {
		_, err = Conn.Exec(query, id)
	} else {
		_, err = tx.Exec(query, id)
	}

	if err != nil {
		return fmt.Errorf("error deleting invite: %v", err)
	}

	return nil
}

func AcceptInvite(invite *Invite, inviteeId string) error {
	// start a transaction
	tx, err := Conn.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}

	// Ensure that rollback is attempted in case of failure
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("transaction rollback error: %v\n", rbErr)
			} else {
				log.Println("transaction rolled back")
			}
		}
	}()

	_, err = tx.Exec(`UPDATE invites SET accepted_at = NOW(), invitee_id = $1 WHERE id = $2`, inviteeId, invite.Id)
	if err != nil {
		return fmt.Errorf("error accepting invite: %v", err)
	}

	// create org user
	err = CreateOrgUser(invite.OrgId, inviteeId, invite.OrgRoleId, tx)

	if err != nil {
		return fmt.Errorf("error creating org user: %v", err)
	}

	// commit transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	invite.InviteeId = &inviteeId

	return nil
}

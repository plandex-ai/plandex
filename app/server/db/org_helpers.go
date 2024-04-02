package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/lib/pq"
	"github.com/plandex/plandex/shared"
)

func GetAccessibleOrgsForUser(user *User) ([]*Org, error) {
	// direct access
	var orgUsers []*OrgUser
	var orgs []*Org

	err := Conn.Select(&orgUsers, "SELECT * FROM orgs_users WHERE user_id = $1", user.Id)

	if err != nil {
		return nil, fmt.Errorf("error getting orgs for user: %v", err)
	}

	orgIds := []string{}
	for _, ou := range orgUsers {
		orgIds = append(orgIds, ou.OrgId)
	}

	if len(orgIds) > 0 {
		err = Conn.Select(&orgs, "SELECT * FROM orgs WHERE id = ANY($1)", pq.Array(orgIds))

		if err != nil {
			return nil, fmt.Errorf("error getting orgs for user: %v", err)
		}
	} else {
		log.Println("No orgs found for user")
		return orgs, nil
	}

	// access via invitation
	invites, err := GetPendingInvitesForEmail(user.Email)

	if err != nil {
		return nil, fmt.Errorf("error getting invites for user: %v", err)
	}

	orgIds = []string{}
	for _, invite := range invites {
		orgIds = append(orgIds, invite.OrgId)
	}

	// log.Println(spew.Sdump(orgIds))

	if len(orgIds) > 0 {
		var orgsFromInvites []*Org
		err = Conn.Select(&orgsFromInvites, "SELECT * FROM orgs WHERE id = ANY($1)", pq.Array(orgIds))
		if err != nil {
			return nil, fmt.Errorf("error getting orgs from invites: %v", err)
		}
		orgs = append(orgs, orgsFromInvites...)
	}

	return orgs, nil
}

func GetOrg(orgId string) (*Org, error) {
	var org Org
	err := Conn.Get(&org, "SELECT * FROM orgs WHERE id = $1", orgId)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("org not found")
		}

		return nil, fmt.Errorf("error getting org: %v", err)
	}

	return &org, nil
}

func ValidateOrgMembership(userId string, orgId string) (bool, error) {
	var count int
	err := Conn.QueryRow("SELECT COUNT(*) FROM orgs_users WHERE user_id = $1 AND org_id = $2", userId, orgId).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("error validating org membership: %v", err)
	}

	return count > 0, nil
}

func CreateOrg(req *shared.CreateOrgRequest, userId string, domain *string, tx *sql.Tx) (*Org, error) {
	org := &Org{
		Name:               req.Name,
		Domain:             domain,
		AutoAddDomainUsers: req.AutoAddDomainUsers,
		OwnerId:            userId,
	}

	err := tx.QueryRow("INSERT INTO orgs (name, domain, auto_add_domain_users, owner_id, is_trial) VALUES ($1, $2, $3, $4, false) RETURNING id", req.Name, domain, req.AutoAddDomainUsers, userId).Scan(&org.Id)

	if err != nil {
		if IsNonUniqueErr(err) {
			// Handle the uniqueness constraint violation
			return nil, fmt.Errorf("an org with domain %s already exists", *domain)

		}

		return nil, fmt.Errorf("error creating org: %v", err)
	}

	orgRoleId, err := GetOrgOwnerRoleId()

	if err != nil {
		return nil, fmt.Errorf("error getting org owner role id: %v", err)
	}

	_, err = tx.Exec("INSERT INTO orgs_users (org_id, user_id, org_role_id) VALUES ($1, $2, $3)", org.Id, userId, orgRoleId)

	if err != nil {
		return nil, fmt.Errorf("error adding org membership: %v", err)
	}

	return org, nil
}

func GetOrgForDomain(domain string) (*Org, error) {
	var org Org
	err := Conn.Get(&org, "SELECT * FROM orgs WHERE domain = $1", domain)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("error getting org for domain: %v", err)
	}

	return &org, nil
}

func AddOrgDomainUsers(orgId, domain string, tx *sql.Tx) error {
	usersForDomain, err := GetUsersForDomain(domain)

	if err != nil {
		return fmt.Errorf("error getting users for domain: %v", err)
	}

	if len(usersForDomain) > 0 {

		// create org users for each user
		var valueStrings []string
		var valueArgs []interface{}
		for i, user := range usersForDomain {
			num := i * 2
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", num+1, num+2))
			valueArgs = append(valueArgs, orgId, user.Id)
		}

		// Join all value strings and execute a single query
		stmt := fmt.Sprintf("INSERT INTO orgs_users (org_id, user_id) VALUES %s", strings.Join(valueStrings, ","))
		_, err = tx.Exec(stmt, valueArgs...)

		if err != nil {
			return fmt.Errorf("error adding org users: %v", err)
		}
	}

	return nil
}

func DeleteOrgUser(orgId, userId string, tx *sql.Tx) error {
	log.Printf("Deleting org user, org: %s | user: %s\n", orgId, userId)

	_, err := tx.Exec("DELETE FROM orgs_users WHERE org_id = $1 AND user_id = $2", orgId, userId)

	if err != nil {
		return fmt.Errorf("error deleting org member: %v", err)
	}

	return nil
}

func CreateOrgUser(orgId, userId, orgRoleId string, tx *sql.Tx) error {
	query := "INSERT INTO orgs_users (org_id, user_id, org_role_id) VALUES ($1, $2, $3)"
	var err error
	if tx == nil {
		_, err = Conn.Exec(query, orgId, userId, orgRoleId)
	} else {
		_, err = tx.Exec(query, orgId, userId, orgRoleId)
	}

	if err != nil {
		return fmt.Errorf("error adding org member: %v", err)
	}

	return nil
}

func ListOrgRoles(orgId string) ([]*OrgRole, error) {
	var orgRoles []*OrgRole
	err := Conn.Select(&orgRoles, "SELECT * FROM org_roles WHERE org_id IS NULL OR org_id = $1", orgId)

	if err != nil {
		return nil, fmt.Errorf("error listing org roles: %v", err)
	}

	return orgRoles, nil
}

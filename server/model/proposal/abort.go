package proposal

import "fmt"

func AbortProposal(proposalId string) error {
	fmt.Println("aborting proposal", proposalId)

	proposal := proposals.Get(proposalId)
	if proposal == nil {
		return fmt.Errorf("proposal not found")
	}

	aborted := proposal.Abort()
	if aborted {
		proposals.Set(proposalId, proposal)
	}

	plan := builds.Get(proposalId)
	if plan != nil {
		aborted := plan.Abort()
		if aborted {
			builds.Set(proposalId, plan)
		}
	}

	return nil
}

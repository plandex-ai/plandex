package proposal

import "fmt"

func AbortProposal(proposalId string) error {
	mu.Lock()
	proposal, ok := proposalsMap[proposalId]

	if !ok {
		mu.Unlock()
		return fmt.Errorf("proposal not found")
	}

	if proposal.Cancel != nil {
		(*proposal.Cancel)()
	}

	proposal.Aborted = true

	mu.Unlock()

	return nil
}

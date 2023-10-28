package types

import (
	"github.com/plandex/plandex/shared"
)

type ConvoSummaryProc struct {
	Err       error
	SummaryCh chan *shared.ConversationSummary
	ErrCh     chan error
}

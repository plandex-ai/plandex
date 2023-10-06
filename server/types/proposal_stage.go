package types

import "context"

type ProposalStage struct {
	CancelFn *context.CancelFunc
	Finished bool
	Err      error
	Aborted  bool
}

func (s *ProposalStage) Abort() bool {
	if s.IsStopped() {
		return false
	}

	if s.CancelFn != nil {
		(*s.CancelFn)()
	}
	s.Aborted = true
	return true
}

func (s *ProposalStage) Finish() bool {
	if s.IsStopped() {
		return false
	}

	if s.CancelFn != nil {
		(*s.CancelFn)()
	}
	s.Finished = true
	return true
}

func (s *ProposalStage) IsStopped() bool {
	return s.Finished || s.Aborted || s.Err != nil
}

func (s *ProposalStage) Cancel() bool {
	if s.IsStopped() {
		return false
	}

	if s.CancelFn == nil {
		return false
	}

	(*s.CancelFn)()

	return true
}

func (s *ProposalStage) SetErr(err error) bool {
	if s.IsStopped() {
		return false
	}

	s.Err = err
	return s.Cancel()
}

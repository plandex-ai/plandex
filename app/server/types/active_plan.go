package types

import (
	"context"
	"plandex-server/db"

	"github.com/google/uuid"
)

type ActiveBuild struct {
	AssistantMessageId string
	ReplyContent       string
	Path               string
	Buffer             string
	Success            bool
	Error              error
	ErrorReason        string
}

type ActivePlan struct {
	Id                  string
	Prompt              string
	StreamCh            chan string
	StreamDoneCh        chan error
	Ctx                 context.Context
	CancelFn            context.CancelFunc
	Contexts            []*db.Context
	ContextsByPath      map[string]*db.Context
	Files               []string
	BuiltFiles          map[string]bool
	CurrentReplyContent string
	NumTokens           int
	PromptMessageNum    int
	BuildQueuesByPath   map[string][]*ActiveBuild
	subscriptions       map[string]chan string
}

func NewActivePlan(planId, prompt string) *ActivePlan {
	ctx, cancel := context.WithCancel(context.Background())

	active := ActivePlan{
		Id:                planId,
		Prompt:            prompt,
		StreamCh:          make(chan string),
		StreamDoneCh:      make(chan error),
		Ctx:               ctx,
		CancelFn:          cancel,
		BuildQueuesByPath: map[string][]*ActiveBuild{},
		Contexts:          []*db.Context{},
		ContextsByPath:    map[string]*db.Context{},
		Files:             []string{},
		BuiltFiles:        map[string]bool{},
		subscriptions:     map[string]chan string{},
	}

	go func() {
		for {
			select {
			case <-active.Ctx.Done():
				return
			case msg := <-active.StreamCh:
				for _, ch := range active.subscriptions {
					ch <- msg
				}
			}
		}
	}()

	return &active
}

func (b *ActiveBuild) BuildFinished() bool {
	return b.Success || b.Error != nil
}

func (ap *ActivePlan) BuildFinished() bool {
	for path := range ap.BuildQueuesByPath {
		if !ap.PathFinished(path) {
			return false
		}
	}
	return true
}

func (ap *ActivePlan) PathFinished(path string) bool {
	for _, build := range ap.BuildQueuesByPath[path] {
		if !build.BuildFinished() {
			return false
		}
	}
	return true
}

func (ap *ActivePlan) Subscribe() (string, chan string) {
	id := uuid.New().String()
	ch := make(chan string)
	ap.subscriptions[id] = ch
	return id, ch
}

func (ap *ActivePlan) Unsubscribe(id string) {
	delete(ap.subscriptions, id)
}

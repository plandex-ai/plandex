package types

import (
	"context"
	"plandex-server/db"

	"github.com/google/uuid"
)

type ActiveBuild struct {
	AssistantMessageId string
	Files              []string
	BuildBuffers       map[string]string
	BuiltFiles         map[string]bool
	Error              error
	ErrorReason        string
}

type ActivePlan struct {
	Prompt            string
	StreamCh          chan string
	StreamDoneCh      chan error
	Ctx               context.Context
	CancelFn          context.CancelFunc
	Contexts          []*db.Context
	ContextsByPath    map[string]*db.Context
	Content           string
	NumTokens         int
	PromptMessageNum  int
	BuildQueuesByPath map[string]*[]ActiveBuild
	subscriptions     map[string]chan string
}

func NewActivePlan(prompt string) *ActivePlan {
	ctx, cancel := context.WithCancel(context.Background())

	active := ActivePlan{
		Prompt:            prompt,
		StreamCh:          make(chan string),
		StreamDoneCh:      make(chan error),
		Ctx:               ctx,
		CancelFn:          cancel,
		BuildQueuesByPath: map[string]*[]ActiveBuild{},
		Contexts:          []*db.Context{},
		ContextsByPath:    map[string]*db.Context{},
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
	return len(b.Files) == len(b.BuiltFiles)
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

package types

import (
	"context"
	"plandex-server/db"

	"github.com/google/uuid"
)

type ActivePlan struct {
	Prompt             string
	StreamCh           chan string
	StreamDoneCh       chan error
	Ctx                context.Context
	CancelFn           context.CancelFunc
	Contexts           []*db.Context
	ContextsByPath     map[string]*db.Context
	Content            string
	NumTokens          int
	PromptMessageNum   int
	AssistantMessageId string
	Files              []string
	BuildBuffers       map[string]string
	BuiltFiles         map[string]bool
	subscriptions      map[string]chan string
}

func NewActivePlan(prompt string) *ActivePlan {
	ctx, cancel := context.WithCancel(context.Background())

	active := ActivePlan{
		Prompt:         prompt,
		StreamCh:       make(chan string),
		StreamDoneCh:   make(chan error),
		Ctx:            ctx,
		CancelFn:       cancel,
		Files:          []string{},
		BuiltFiles:     map[string]bool{},
		Contexts:       []*db.Context{},
		ContextsByPath: map[string]*db.Context{},
		BuildBuffers:   map[string]string{},
		subscriptions:  map[string]chan string{},
	}

	go func() {
		for {
			select {
			case <-active.Ctx.Done():
				return
			case msg := <-active.StreamCh:
				for _, ch := range active.subscriptions {
					go func(ch chan string) {
						ch <- msg
					}(ch)
				}
			}
		}
	}()

	return &active
}

func (ap *ActivePlan) BuildFinished() bool {
	return len(ap.Files) == len(ap.BuiltFiles)
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

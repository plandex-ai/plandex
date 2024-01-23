package types

import (
	"context"
	"encoding/json"
	"plandex-server/db"

	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
)

type ActiveBuild struct {
	AssistantMessageId string
	ReplyContent       string
	FileContent        string
	Path               string
	Buffer             string
	Success            bool
	Error              error
}

type ActivePlan struct {
	Id                  string
	Prompt              string
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
	StreamDoneCh        chan error

	streamCh      chan string
	subscriptions map[string]chan string
}

func NewActivePlan(planId, prompt string) *ActivePlan {
	ctx, cancel := context.WithCancel(context.Background())

	active := ActivePlan{
		Id:                planId,
		Prompt:            prompt,
		Ctx:               ctx,
		CancelFn:          cancel,
		BuildQueuesByPath: map[string][]*ActiveBuild{},
		Contexts:          []*db.Context{},
		ContextsByPath:    map[string]*db.Context{},
		Files:             []string{},
		BuiltFiles:        map[string]bool{},

		streamCh:      make(chan string),
		StreamDoneCh:  make(chan error),
		subscriptions: map[string]chan string{},
	}

	go func() {
		for {
			select {
			case <-active.Ctx.Done():
				return
			case msg := <-active.streamCh:
				for _, ch := range active.subscriptions {
					ch <- msg
				}
			}
		}
	}()

	return &active
}

func (ap *ActivePlan) Stream(msg shared.StreamMessage) {
	msgJson, err := json.Marshal(msg)
	if err != nil {
		ap.StreamDoneCh <- err
		return
	}
	ap.streamCh <- string(msgJson)
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

func (b *ActiveBuild) BuildFinished() bool {
	return b.Success || b.Error != nil
}

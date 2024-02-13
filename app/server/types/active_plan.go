package types

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
)

type ActiveBuild struct {
	ReplyId      string
	ReplyContent string
	FileContent  string
	Path         string
	Buffer       string
	Success      bool
	Error        error
}

type ActivePlan struct {
	Id                      string
	CurrentStreamingReplyId string
	CurrentReplyDoneCh      chan bool
	Branch                  string
	Prompt                  string
	BuildOnly               bool
	Ctx                     context.Context
	CancelFn                context.CancelFunc
	ModelStreamCtx          context.Context
	CancelModelStreamFn     context.CancelFunc
	Contexts                []*db.Context
	ContextsByPath          map[string]*db.Context
	Files                   []string
	BuiltFiles              map[string]bool
	IsBuildingByPath        map[string]bool
	CurrentReplyContent     string
	NumTokens               int
	PromptMessageNum        int
	BuildQueuesByPath       map[string][]*ActiveBuild
	RepliesFinished         bool
	StreamDoneCh            chan *shared.ApiError
	ModelStreamId           string
	IsBackground            bool
	MissingFileResponseCh   chan shared.RespondMissingFileChoice
	AllowOverwritePaths     map[string]bool
	SkippedPaths            map[string]bool
	streamCh                chan string
	subscriptions           map[string]chan string
}

func NewActivePlan(planId, branch, prompt string, buildOnly bool) *ActivePlan {
	ctx, cancel := context.WithCancel(context.Background())
	// child context for model stream so we can cancel it separately if needed
	modelStreamCtx, cancelModelStream := context.WithCancel(ctx)

	active := ActivePlan{
		Id:                    planId,
		BuildOnly:             buildOnly,
		Branch:                branch,
		Prompt:                prompt,
		Ctx:                   ctx,
		CancelFn:              cancel,
		ModelStreamCtx:        modelStreamCtx,
		CancelModelStreamFn:   cancelModelStream,
		BuildQueuesByPath:     map[string][]*ActiveBuild{},
		Contexts:              []*db.Context{},
		ContextsByPath:        map[string]*db.Context{},
		Files:                 []string{},
		BuiltFiles:            map[string]bool{},
		IsBuildingByPath:      map[string]bool{},
		StreamDoneCh:          make(chan *shared.ApiError),
		MissingFileResponseCh: make(chan shared.RespondMissingFileChoice),
		AllowOverwritePaths:   map[string]bool{},
		SkippedPaths:          map[string]bool{},
		streamCh:              make(chan string),
		subscriptions:         map[string]chan string{},
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
		ap.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error marshalling stream message: " + err.Error(),
		}
		return
	}
	ap.streamCh <- string(msgJson)

	if msg.Type == shared.StreamMessageFinished {
		// Wait briefly allow last stream message to be sent
		time.Sleep(100 * time.Millisecond)
		ap.StreamDoneCh <- nil
	}
}

func (ap *ActivePlan) ResetModelCtx() {
	ap.ModelStreamCtx, ap.CancelModelStreamFn = context.WithCancel(ap.Ctx)
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

func (ap *ActivePlan) NumSubscribers() int {
	return len(ap.subscriptions)
}

func (b *ActiveBuild) BuildFinished() bool {
	return b.Success || b.Error != nil
}

func (ap *ActivePlan) PendingBuildsByPath(orgId, userId string, convoMessagesArg []*db.ConvoMessage) (map[string][]*ActiveBuild, error) {
	var planDescs []*db.ConvoMessageDescription
	var currentPlan *shared.CurrentPlanState
	var convoMessagesById map[string]*db.ConvoMessage

	errCh := make(chan error)

	go func() {
		var err error
		planDescs, err = db.GetPendingBuildDescriptions(orgId, ap.Id)
		if err != nil {
			errCh <- fmt.Errorf("error getting pending build descriptions: %v", err)
			return
		}

		currentPlan, err = db.GetCurrentPlanState(db.CurrentPlanStateParams{
			OrgId:                    orgId,
			PlanId:                   ap.Id,
			PendingBuildDescriptions: planDescs,
		})
		if err != nil {
			errCh <- fmt.Errorf("error getting current plan state: %v", err)
			return
		}

		errCh <- nil
	}()

	go func() {
		var convoMessages []*db.ConvoMessage
		if convoMessagesArg == nil {
			var err error
			convoMessages, err = db.GetPlanConvo(orgId, ap.Id)

			if err != nil {
				errCh <- fmt.Errorf("error getting plan convo: %v", err)
				return
			}
		} else {
			convoMessages = convoMessagesArg
		}

		convoMessagesById = map[string]*db.ConvoMessage{}
		for _, msg := range convoMessages {
			convoMessagesById[msg.Id] = msg
		}

		errCh <- nil
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil {
			log.Printf("Error getting plan data: %v\n", err)
			return nil, err
		}
	}

	log.Println("planDescs:")
	spew.Dump(planDescs)

	activeBuildsByPath := map[string][]*ActiveBuild{}

	for _, desc := range planDescs {
		if !desc.DidBuild && len(desc.Files) > 0 {
			if desc.ConvoMessageId == "" {
				log.Printf("No convo message ID for description: %v\n", desc)
				return nil, fmt.Errorf("no convo message ID for description: %v", desc)
			}

			if convoMessagesById[desc.ConvoMessageId] == nil {
				log.Printf("No convo message for ID: %s\n", desc.ConvoMessageId)
				return nil, fmt.Errorf("no convo message for ID: %s", desc.ConvoMessageId)
			}

			for _, file := range desc.Files {
				if activeBuildsByPath[file] == nil {
					activeBuildsByPath[file] = []*ActiveBuild{}
				}

				activeBuildsByPath[file] = append(activeBuildsByPath[file], &ActiveBuild{
					ReplyId:      desc.ConvoMessageId,
					FileContent:  currentPlan.CurrentPlanFiles.Files[file],
					Path:         file,
					ReplyContent: convoMessagesById[desc.ConvoMessageId].Message,
				})
			}
		}
	}

	log.Println("activeBuildsByPath:")
	spew.Dump(activeBuildsByPath)

	return activeBuildsByPath, nil
}

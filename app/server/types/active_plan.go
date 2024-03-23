package types

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"plandex-server/db"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
)

type ActiveBuild struct {
	ReplyId           string
	FileDescription   string
	FileContent       string
	FileContentTokens int
	CurrentFileTokens int
	Path              string
	Idx               int
	Buffer            string
	BufferTokens      int
	Success           bool
	Error             error
}

type subscription struct {
	ch           chan string
	ctx          context.Context
	cancelFn     context.CancelFunc
	mu           sync.Mutex // Protects the messageQueue
	messageQueue []string
	cond         *sync.Cond // Used to wait for and signal new messages
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
	SummaryCtx              context.Context
	SummaryCancelFn         context.CancelFunc
	Contexts                []*db.Context
	ContextsByPath          map[string]*db.Context
	Files                   []string
	BuiltFiles              map[string]bool
	IsBuildingByPath        map[string]bool
	CurrentReplyContent     string
	NumTokens               int
	MessageNum              int
	BuildQueuesByPath       map[string][]*ActiveBuild
	RepliesFinished         bool
	StreamDoneCh            chan *shared.ApiError
	ModelStreamId           string
	MissingFilePath         string
	MissingFileResponseCh   chan shared.RespondMissingFileChoice
	AllowOverwritePaths     map[string]bool
	SkippedPaths            map[string]bool
	StoredReplyIds          []string
	streamCh                chan string
	subscriptions           map[string]*subscription
	subscriptionMu          sync.Mutex
}

func NewActivePlan(planId, branch, prompt string, buildOnly bool) *ActivePlan {
	ctx, cancel := context.WithCancel(context.Background())
	// child context for model stream so we can cancel it separately if needed
	modelStreamCtx, cancelModelStream := context.WithCancel(ctx)

	// we don't want to cancel summaries unless the whole plan is stopped or there's an error -- if the active plan finishes, we want summaries to continue -- so they get their own context
	summaryCtx, cancelSummary := context.WithCancel(context.Background())

	active := ActivePlan{
		Id:                    planId,
		BuildOnly:             buildOnly,
		Branch:                branch,
		Prompt:                prompt,
		Ctx:                   ctx,
		CancelFn:              cancel,
		ModelStreamCtx:        modelStreamCtx,
		CancelModelStreamFn:   cancelModelStream,
		SummaryCtx:            summaryCtx,
		SummaryCancelFn:       cancelSummary,
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
		subscriptions:         map[string]*subscription{},
		subscriptionMu:        sync.Mutex{},
	}

	go func() {
		defer func() {
			log.Println("ActivePlan stream manager returned")
			if r := recover(); r != nil {
				log.Printf("Recovered in send to subscriber: %v\n", r)
			}
		}()
		for {
			select {
			case <-active.Ctx.Done():
				return
			case msg := <-active.streamCh:
				var subscriptions map[string]*subscription
				active.subscriptionMu.Lock()
				subscriptions = active.subscriptions
				active.subscriptionMu.Unlock()
				for _, sub := range subscriptions {
					sub.enqueueMessage(msg)
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

	// log.Printf("ActivePlan: sending stream message: %s\n", string(msgJson))

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
	ap.subscriptionMu.Lock()
	defer ap.subscriptionMu.Unlock()
	id := uuid.New().String()
	sub := newSubscription()
	ap.subscriptions[id] = sub
	return id, sub.ch
}

func (ap *ActivePlan) Unsubscribe(id string) {
	ap.subscriptionMu.Lock()
	defer ap.subscriptionMu.Unlock()

	sub, ok := ap.subscriptions[id]

	if ok {
		sub.cancelFn()
		sub.cond.Signal()
		delete(ap.subscriptions, id)
	}
}

func (ap *ActivePlan) NumSubscribers() int {
	ap.subscriptionMu.Lock()
	defer ap.subscriptionMu.Unlock()
	return len(ap.subscriptions)
}

func (b *ActiveBuild) BuildFinished() bool {
	return b.Success || b.Error != nil
}

func newSubscription() *subscription {
	ctx, cancel := context.WithCancel(context.Background())
	sub := &subscription{
		ch:           make(chan string),
		ctx:          ctx,
		cancelFn:     cancel,
		messageQueue: make([]string, 0),
	}
	sub.mu = sync.Mutex{}
	sub.cond = sync.NewCond(&sub.mu)
	go sub.processMessages()
	return sub
}

func (sub *subscription) processMessages() {
	for {
		sub.mu.Lock()
		for len(sub.messageQueue) == 0 {
			sub.cond.Wait()           // Automatically unlocks sub.mu and waits; re-locks sub.mu upon waking.
			if sub.ctx.Err() != nil { // Check if context is cancelled after waking up.
				sub.mu.Unlock()
				return
			}
		}
		// At this point, there is at least one message in the queue
		msg := sub.messageQueue[0]
		sub.messageQueue = sub.messageQueue[1:]
		sub.mu.Unlock()

		select {
		case <-sub.ctx.Done():
			log.Println("ActivePlan: subscription context done, aborting send")
			return
		case sub.ch <- msg:
			// Message sent, proceed to next
		}
	}
}

// Adding a message to the subscription's queue
func (sub *subscription) enqueueMessage(msg string) {
	// log.Printf("ActivePlan: enqueueing message: %s\n", msg)
	sub.mu.Lock()
	sub.messageQueue = append(sub.messageQueue, msg)
	sub.mu.Unlock()
	sub.cond.Signal() // Signal the waiting goroutine that a new message is available
}

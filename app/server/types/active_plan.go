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

const MaxStreamRate = 50 * time.Millisecond

// const MaxConcurrentBuildStreams = 3 // otherwise we get EOF errors from openai

type ActiveBuild struct {
	ReplyId                  string
	FileDescription          string
	FileContent              string
	FileContentTokens        int
	CurrentFileTokens        int
	Path                     string
	Idx                      int
	WithLineNumsBuffer       string
	WithLineNumsBufferTokens int
	VerifyBuffer             string
	VerifyBufferTokens       int
	FixBuffer                string
	FixBufferTokens          int
	Success                  bool
	Error                    error
	IsVerification           bool
	ToVerifyUpdatedState     string
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
	UserId                  string
	OrgId                   string
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
	LatestSummaryCh         chan *db.ConvoSummary
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

	subscriptions  map[string]*subscription
	subscriptionMu sync.Mutex

	streamCh              chan string
	streamMu              sync.Mutex
	lastStreamMessageSent time.Time
	streamMessageBuffer   []shared.StreamMessage
}

func NewActivePlan(orgId, userId, planId, branch, prompt string, buildOnly bool) *ActivePlan {
	ctx, cancel := context.WithCancel(context.Background())
	// child context for model stream so we can cancel it separately if needed
	modelStreamCtx, cancelModelStream := context.WithCancel(ctx)

	// we don't want to cancel summaries unless the whole plan is stopped or there's an error -- if the active plan finishes, we want summaries to continue -- so they get their own context
	summaryCtx, cancelSummary := context.WithCancel(context.Background())

	active := ActivePlan{
		Id:                    planId,
		OrgId:                 orgId,
		UserId:                userId,
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

func (ap *ActivePlan) FlushStreamBuffer() {
	// log.Println("ActivePlan: flush stream buffer")

	ap.streamMu.Lock()
	if len(ap.streamMessageBuffer) == 0 {
		// log.Println("ActivePlan: stream buffer empty")
		ap.streamMu.Unlock()
		return
	}

	// log.Printf("ActivePlan: flushing %d messages from stream buffer\n", len(ap.streamMessageBuffer))

	var msg shared.StreamMessage
	if len(ap.streamMessageBuffer) == 1 {
		log.Println("ActivePlan: flushing 1 message from stream buffer")
		msg = ap.streamMessageBuffer[0]
	} else {
		msg = shared.StreamMessage{
			Type:           shared.StreamMessageMulti,
			StreamMessages: ap.streamMessageBuffer,
		}
	}

	ap.streamMessageBuffer = []shared.StreamMessage{}

	ap.streamMu.Unlock()

	ap.Stream(msg)
}

func (ap *ActivePlan) Stream(msg shared.StreamMessage) {
	// log.Printf("ActivePlan: received Stream message: %v\n", msg)

	ap.streamMu.Lock()
	defer ap.streamMu.Unlock()

	if msg.Type != shared.StreamMessageFinished &&
		msg.Type != shared.StreamMessagePromptMissingFile {
		if time.Since(ap.lastStreamMessageSent) < MaxStreamRate {
			// log.Println("ActivePlan: stream rate limiting -- buffering message")
			ap.streamMessageBuffer = append(ap.streamMessageBuffer, msg)
			return
		} else if len(ap.streamMessageBuffer) > 0 {
			// log.Println("ActivePlan: stream buffer not empty -- flushing buffer before sending message")
			ap.streamMessageBuffer = append(ap.streamMessageBuffer, msg)

			// unlock before recursive call
			ap.streamMu.Unlock()
			ap.FlushStreamBuffer()
			ap.streamMu.Lock() // re-lock after recursive call
			return
		}
	}

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

	if msg.Type == shared.StreamMessageFinished {
		// send full buffer if we got a finished message
		if len(ap.streamMessageBuffer) > 0 {
			// log.Println("ActivePlan: finished message -- sending stream buffer first")

			// unlock before recursive call
			ap.streamMu.Unlock()
			ap.FlushStreamBuffer()
			ap.streamMu.Lock() // re-lock after recursive call

			// sleep a little before the final message
			time.Sleep(50 * time.Millisecond)
		}
	}

	// log.Println("ActivePlan: sending stream message:", msg.Type)

	ap.streamCh <- string(msgJson)

	// log.Println("ActivePlan: sent stream message:", msg.Type)

	now := time.Now()
	if now.After(ap.lastStreamMessageSent) {
		ap.lastStreamMessageSent = now
	}

	if msg.Type == shared.StreamMessageFinished {
		// Wait briefly to allow last stream message to be sent
		time.Sleep(100 * time.Millisecond)
		ap.StreamDoneCh <- nil
	}
}

func (ap *ActivePlan) ResetModelCtx() {
	ap.ModelStreamCtx, ap.CancelModelStreamFn = context.WithCancel(ap.Ctx)
}

func (ap *ActivePlan) BuildFinished() bool {
	for path := range ap.BuildQueuesByPath {
		if ap.IsBuildingByPath[path] || !ap.PathQueueEmpty(path) {
			return false
		}
	}
	return true
}

func (ap *ActivePlan) PathQueueEmpty(path string) bool {
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

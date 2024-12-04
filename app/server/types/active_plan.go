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

const MaxStreamRate = 70 * time.Millisecond

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
	AutoContext             bool
	AutoLoadContextCh       chan struct{}
	AllowOverwritePaths     map[string]bool
	SkippedPaths            map[string]bool
	StoredReplyIds          []string
	DidEditFiles            bool
	DidVerifyDiff           bool
	IsVerifyingDiff         bool

	subscriptions  map[string]*subscription
	subscriptionMu sync.Mutex

	streamCh              chan string
	streamMu              sync.Mutex
	lastStreamMessageSent time.Time
	streamMessageBuffer   []shared.StreamMessage
}

func NewActivePlan(orgId, userId, planId, branch, prompt string, buildOnly, autoContext bool) *ActivePlan {
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
		AutoContext:           autoContext,
		AutoLoadContextCh:     make(chan struct{}),
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
	ap.streamMu.Lock()
	if len(ap.streamMessageBuffer) == 0 {
		ap.streamMu.Unlock()
		return
	}

	bufferToFlush := ap.streamMessageBuffer
	ap.streamMessageBuffer = []shared.StreamMessage{}
	ap.streamMu.Unlock()

	if len(bufferToFlush) == 1 {
		log.Println("ActivePlan.FlushStreamBuffer: flushing single message")
		ap.Stream(bufferToFlush[0])
	} else {
		log.Println("ActivePlan.FlushStreamBuffer: flushing multi-message")
		ap.Stream(shared.StreamMessage{
			Type:           shared.StreamMessageMulti,
			StreamMessages: bufferToFlush,
		})
	}
}

func (ap *ActivePlan) Stream(msg shared.StreamMessage) {
	// log.Println("ActivePlan.Stream:")
	// log.Println(msg)

	ap.streamMu.Lock()

	skipBuffer := false
	if msg.Type == shared.StreamMessagePromptMissingFile || msg.Type == shared.StreamMessageFinished {
		skipBuffer = true
	}

	// Special messages bypass buffering
	if !skipBuffer {
		// log.Println("ActivePlan.Stream: time since last message sent:", time.Since(ap.lastStreamMessageSent))

		if time.Since(ap.lastStreamMessageSent) < MaxStreamRate {
			// log.Println("ActivePlan.Stream: buffering message")

			// Buffer the message
			ap.streamMessageBuffer = append(ap.streamMessageBuffer, msg)
			ap.streamMu.Unlock()
			return
		} else if len(ap.streamMessageBuffer) > 0 {
			// log.Println("ActivePlan.Stream: flushing buffer")

			// Need to flush buffer first
			ap.streamMessageBuffer = append(ap.streamMessageBuffer, msg)
			bufferToFlush := ap.streamMessageBuffer
			ap.streamMessageBuffer = []shared.StreamMessage{}
			ap.streamMu.Unlock()

			// log.Println("ActivePlan.Stream: sending multi-message:")
			// log.Println(bufferToFlush)

			// Send as multi-message
			ap.Stream(shared.StreamMessage{
				Type:           shared.StreamMessageMulti,
				StreamMessages: bufferToFlush,
			})
			return
		}
	}

	// Direct send path
	msgJson, err := json.Marshal(msg)
	if err != nil {
		ap.streamMu.Unlock()
		ap.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error marshalling stream message: " + err.Error(),
		}
		return
	}

	if skipBuffer && len(ap.streamMessageBuffer) > 0 {
		// Handle any remaining buffered messages before sending the message
		// log.Println("ActivePlan.Stream: message is a skip buffer type and there are buffered messages")
		// log.Println("ActivePlan.Stream: flushing remaining buffered messages before skip buffer message is sent")
		bufferToFlush := ap.streamMessageBuffer
		ap.streamMessageBuffer = []shared.StreamMessage{}
		ap.streamMu.Unlock()

		log.Println("Flushing buffered messages before finishing")
		// log.Println("ActivePlan.Stream: sending multi-message:")
		// log.Println(bufferToFlush)
		ap.Stream(shared.StreamMessage{
			Type:           shared.StreamMessageMulti,
			StreamMessages: bufferToFlush,
		})
		log.Println("ActivePlan.Stream: finished flushing buffered messages. waiting 50ms before sending skip buffer type message")
		time.Sleep(50 * time.Millisecond)
		log.Println("ActivePlan.Stream: sending finish message")
		ap.Stream(msg) // send the skip buffer type message

		ap.streamMu.Lock()
		now := time.Now()
		if now.After(ap.lastStreamMessageSent) {
			ap.lastStreamMessageSent = now
		}
		ap.streamMu.Unlock()

		if msg.Type == shared.StreamMessageFinished {
			// wait for the finish message to be sent then send the done signal
			log.Println("ActivePlan.Stream: waiting 50ms before sending done signal")
			time.Sleep(50 * time.Millisecond)
			log.Println("ActivePlan.Stream: sending done signal")
			ap.StreamDoneCh <- nil
			return
		}
	}

	// log.Println("ActivePlan.Stream: sending direct message")
	// log.Println(string(msgJson))

	ap.streamCh <- string(msgJson)

	now := time.Now()
	if now.After(ap.lastStreamMessageSent) {
		ap.lastStreamMessageSent = now
	}
	ap.streamMu.Unlock()

	if msg.Type == shared.StreamMessageFinished {
		log.Println("ActivePlan.Stream: waiting 50ms before sending done signal")
		time.Sleep(50 * time.Millisecond)
		log.Println("ActivePlan.Stream: sending done signal")
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

func (ap *ActivePlan) ShouldVerifyDiff() bool {
	// disabling tell-based verifications for now
	return false

	// log.Printf("ShouldVerifyDiff: buildOnly=%v, didVerifyDiff=%v, numBuiltFiles=%d, didEditFiles=%v, hasMultipleBuiltFiles=%v",
	// 	!ap.BuildOnly,
	// 	!ap.DidVerifyDiff,
	// 	len(ap.BuiltFiles),
	// 	ap.DidEditFiles,
	// 	len(ap.BuiltFiles) > 3)

	// return !ap.BuildOnly &&
	// 	!ap.IsVerifyingDiff &&
	// 	!ap.DidVerifyDiff &&
	// 	(len(ap.BuiltFiles) > 0 || len(ap.IsBuildingByPath) > 0) &&
	// 	(ap.DidEditFiles || len(ap.BuiltFiles) > 3) // verify diff if we edited any files or built more than 3 new files—unlikely to have errors otherwise
}

func (ap *ActivePlan) Finish() {
	ap.Stream(shared.StreamMessage{
		Type: shared.StreamMessageFinished,
	})
}

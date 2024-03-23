package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	modelPlan "plandex-server/model/plan"
	"plandex-server/types"
	"time"

	"github.com/plandex/plandex/shared"
)

func startResponseStream(w http.ResponseWriter, auth *types.ServerAuth, planId, branch string, isConnect bool) {
	log.Println("Response stream manager: starting plan stream")

	active := modelPlan.GetActivePlan(planId, branch)

	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// send initial message to client
	msg := shared.StreamMessage{
		Type: shared.StreamMessageStart,
	}

	bytes, err := json.Marshal(msg)

	if err != nil {
		log.Printf("Response stream manager: error marshalling message: %v\n", err)
		return
	}

	log.Println("Response stream manager: sending initial message")
	err = sendStreamMessage(w, string(bytes))
	if err != nil {
		log.Println("Response stream manager: error sending initial message:", err)
		return
	}

	if isConnect {
		time.Sleep(100 * time.Millisecond)
		err = initConnectActive(auth, planId, branch, w)

		if err != nil {
			log.Println("Response stream manager: error initializing connection to active plan:", err)
			return
		}
	}

	subscriptionId, ch := modelPlan.SubscribePlan(planId, branch)
	defer func() {
		log.Println("Response stream manager: client stream closed")
		modelPlan.UnsubscribePlan(planId, branch, subscriptionId)
	}()

	if isConnect {
		time.Sleep(50 * time.Millisecond)
	} else {
		time.Sleep(100 * time.Millisecond)
	}

	for {
		select {
		case <-active.Ctx.Done():
			log.Println("Response stream manager: context done")
			return
		case msg := <-ch:
			// log.Println("Response stream manager: sending message:", msg)
			err = sendStreamMessage(w, msg)
			if err != nil {
				return
			}
		}
	}

}

func sendStreamMessage(w http.ResponseWriter, msg string) error {
	bytes := []byte(msg + shared.STREAM_MESSAGE_SEPARATOR)

	// log.Printf("Response stream manager: writing message to client: %s\n", msg)

	_, err := w.Write(bytes)
	if err != nil {
		log.Printf("Response stream manager: error writing to client: %v\n", err)
		return err
	} else if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func initConnectActive(auth *types.ServerAuth, planId, branch string, w http.ResponseWriter) error {
	log.Println("Response stream manager: initializing connection to active plan")

	active := modelPlan.GetActivePlan(planId, branch)

	msg := shared.StreamMessage{
		Type: shared.StreamMessageConnectActive,
	}

	if active.Prompt != "" && !active.BuildOnly {
		msg.InitPrompt = active.Prompt
	}

	if active.BuildOnly {
		msg.InitBuildOnly = true
	}

	if len(active.StoredReplyIds) > 0 {
		convo, err := db.GetPlanConvo(auth.OrgId, active.Id)
		if err != nil {
			return fmt.Errorf("error getting plan convo: %v", err)
		}

		convoMsgById := map[string]*db.ConvoMessage{}
		for _, convoMsg := range convo {
			convoMsgById[convoMsg.Id] = convoMsg
		}

		for _, replyId := range active.StoredReplyIds {
			if convoMsg, ok := convoMsgById[replyId]; ok {
				msg.InitReplies = append(msg.InitReplies, convoMsg.Message)
			}
		}
	}

	if active.CurrentReplyContent != "" {
		msg.InitReplies = append(msg.InitReplies, active.CurrentReplyContent)
	}

	if active.MissingFilePath != "" {
		msg.MissingFilePath = active.MissingFilePath
	}

	bytes, err := json.Marshal(msg)

	if err != nil {
		return fmt.Errorf("error marshalling message: %v", err)
	}

	log.Println("Response stream manager: sending connect message")
	err = sendStreamMessage(w, string(bytes))

	if err != nil {
		return fmt.Errorf("error sending connect message: %v", err)
	}

	// if we're connecting to an active stream and there are active builds, send initial build info
	if len(active.BuildQueuesByPath) > 0 {

		for path, queue := range active.BuildQueuesByPath {
			buildInfo := shared.BuildInfo{Path: path}

			for _, build := range queue {
				if build.BuildFinished() {
					buildInfo.NumTokens = 0
					buildInfo.Finished = true
				} else {
					tokens := build.BufferTokens

					buildInfo.Finished = false
					buildInfo.NumTokens += tokens
				}
			}

			msg := shared.StreamMessage{
				Type:      shared.StreamMessageBuildInfo,
				BuildInfo: &buildInfo,
			}
			bytes, err := json.Marshal(msg)

			if err != nil {
				return fmt.Errorf("error marshalling message: %v", err)
			}

			err = sendStreamMessage(w, string(bytes))

			if err != nil {
				return fmt.Errorf("error sending message: %v", err)
			}

		}

	}

	return nil
}

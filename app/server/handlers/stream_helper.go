package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
)

func startResponseStream(w http.ResponseWriter, ch chan string, active *types.ActivePlan, closeFn func()) {
	log.Println("Response stream manager: starting plan stream")

	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	defer func() {
		log.Println("Response stream manager: plan stream done")
		closeFn()
	}()

	// send initial message to client
	msg := shared.StreamMessage{
		Type: shared.StreamMessageStart,
	}
	bytes, err := json.Marshal(msg)

	if err != nil {
		log.Printf("Response stream manager: error marshalling message: %v\n", err)
		return
	}

	err = sendStreamMessage(w, active, string(bytes))
	if err != nil {
		return
	}

	for {
		select {
		case <-active.Ctx.Done():
			log.Println("Response stream manager: context done")
			return
		case msg := <-ch:
			// log.Println("Response stream manager: sending message:", msg)
			err = sendStreamMessage(w, active, msg)
			if err != nil {
				return
			}
		}
	}

}

func sendStreamMessage(w http.ResponseWriter, active *types.ActivePlan, msg string) error {
	bytes := []byte(msg + shared.STREAM_MESSAGE_SEPARATOR)
	_, err := w.Write(bytes)
	if err != nil {
		log.Printf("Response stream mananger: error writing to client: %v\n", err)
		if !active.IsBackground && active.NumSubscribers() == 1 {
			active.CancelFn()
		}
		return err
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

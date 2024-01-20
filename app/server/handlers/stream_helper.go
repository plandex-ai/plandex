package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/plandex/plandex/shared"
)

func startResponseStream(w http.ResponseWriter, ch chan string, ctx context.Context, closeFn func()) {
	log.Println("Response stream manager: starting plan stream")

	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	defer func() {
		log.Println("Response stream manager: plan stream done")
		closeFn()
	}()

	for {
		select {
		case <-ctx.Done():
			log.Println("Response stream manager: context done")
			return
		case msg := <-ch:
			// log.Println("Response stream manager: sending message:", msg)

			bytes := []byte(msg + shared.STREAM_MESSAGE_SEPARATOR)
			_, err := w.Write(bytes)
			if err != nil {
				log.Printf("Response stream mananger: error writing to client: %v\n", err)
				return
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}

}

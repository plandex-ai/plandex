package handlers

import (
	"fmt"
	"log"
	"net/http"
	"plandex-server/types"
	"sync"

	"github.com/plandex/plandex/shared"
)

func startStream(w http.ResponseWriter, startFn func(onStream types.OnStreamFunc) error) {
	var isResponseClosed bool
	done := make(chan struct{})
	var streamMu sync.Mutex

	onStream := func(content string, err error) {
		streamMu.Lock()
		defer streamMu.Unlock()

		if isResponseClosed {
			return
		}

		if err != nil {
			isResponseClosed = true
			log.Printf("Error writing stream content to client: %v\n", err)
			close(done)
			return
		}

		// stream content to client
		if content != "" {
			// fmt.Println("writing stream content to client:", content)

			bytes := []byte(content + shared.STREAM_MESSAGE_SEPARATOR)
			_, err = w.Write(bytes)
			if err != nil {
				isResponseClosed = true
				log.Printf("Error writing stream content to client: %v\n", err)
				log.Printf("Content: %s\n", string(bytes))
				close(done)
				return
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}

		if content == shared.STREAM_FINISHED {
			isResponseClosed = true
			close(done)
			fmt.Println("proposal stream finished")
			return
		}
	}

	err := startFn(onStream)
	if err != nil {
		log.Printf("Error starting stream: %v\n", err)
		close(done)
		return
	}

	// block until streaming is done
	<-done

	fmt.Println("done")

}

package api

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/plandex/plandex/shared"
)

func connectPlanRespStream(body io.ReadCloser, onStream OnStreamPlan) {
	streamState := shared.NewPlanStreamState()
	reader := bufio.NewReader(body)

	go func() {
		for {
			s, err := readUntilSeparator(reader, shared.STREAM_MESSAGE_SEPARATOR)
			if err != nil {
				fmt.Println("Error reading line:", err)
				streamState.Event(context.Background(), shared.EVENT_ERROR)
				onStream(OnStreamPlanParams{Content: "", State: streamState, Err: err})
				body.Close()
				return
			}

			if s == shared.STREAM_FINISHED || s == shared.STREAM_ABORTED {
				var evt string
				if s == shared.STREAM_FINISHED {
					evt = shared.EVENT_FINISH
				} else {
					evt = shared.STATE_ABORTED
				}
				err := streamState.Event(context.Background(), evt)
				if err != nil {
					fmt.Printf("Error triggering state change %s: %s\n", evt, err)
				}
				onStream(OnStreamPlanParams{Content: "", State: streamState, Err: err})
				body.Close()
				return
			}

			if s == shared.STREAM_DESCRIPTION_PHASE {
				err = streamState.Event(context.Background(), shared.EVENT_DESCRIBE)
			} else if s == shared.STREAM_BUILD_PHASE {
				err = streamState.Event(context.Background(), shared.EVENT_BUILD)
			} else if s == shared.STREAM_WRITE_PHASE {
				err = streamState.Event(context.Background(), shared.EVENT_WRITE)
			}

			if err != nil {
				fmt.Println("Error setting state:", err)
				onStream(OnStreamPlanParams{Content: "", State: streamState, Err: err})
				body.Close()
				return
			}

			onStream(OnStreamPlanParams{Content: s, State: streamState, Err: nil})
		}
	}()
}

func readUntilSeparator(reader *bufio.Reader, separator string) (string, error) {
	var result []byte
	sepBytes := []byte(separator)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return string(result), err
		}
		result = append(result, b)
		if len(result) >= len(sepBytes) && bytes.HasSuffix(result, sepBytes) {
			return string(result[:len(result)-len(separator)]), nil
		}
	}
}

package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"plandex-cli/types"
	"time"

	shared "plandex-shared"
)

// 3 heartbeat misses = timeout
const HeartbeatTimeout = 16 * time.Second

func connectPlanRespStream(body io.ReadCloser, onStream types.OnStreamPlan) {
	reader := bufio.NewReader(body)
	timer := time.NewTimer(HeartbeatTimeout)
	defer timer.Stop()

	go func() {
		for {
			select {
			case <-timer.C:
				log.Println("Connection to plan stream timed out due to missing heartbeats")
				onStream(types.OnStreamPlanParams{Msg: nil, Err: fmt.Errorf("connection to plan stream timed out due to missing heartbeats")})
				body.Close()
				return
			default:
			}

			s, err := readUntilSeparator(reader, shared.STREAM_MESSAGE_SEPARATOR)
			if err != nil {
				log.Println("Error reading line:", err)
				onStream(types.OnStreamPlanParams{Msg: nil, Err: err})
				body.Close()
				return
			}

			timer.Reset(HeartbeatTimeout)

			// ignore heartbeats
			if s == string(shared.StreamMessageHeartbeat) {
				continue
			}

			var msg shared.StreamMessage
			err = json.Unmarshal([]byte(s), &msg)
			if err != nil {
				log.Println("Error unmarshalling message:", err)
				onStream(types.OnStreamPlanParams{Msg: nil, Err: err})
				body.Close()
				return
			}

			// log.Println("connectPlanRespStream: received message:", msg)

			onStream(types.OnStreamPlanParams{Msg: &msg, Err: nil})

			if msg.Type == shared.StreamMessageFinished || msg.Type == shared.StreamMessageError || msg.Type == shared.StreamMessageAborted {
				body.Close()
				return
			}

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

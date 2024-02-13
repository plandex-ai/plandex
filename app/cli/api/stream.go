package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"plandex/types"

	"github.com/plandex/plandex/shared"
)

func connectPlanRespStream(body io.ReadCloser, onStream types.OnStreamPlan) {
	reader := bufio.NewReader(body)

	go func() {
		for {
			s, err := readUntilSeparator(reader, shared.STREAM_MESSAGE_SEPARATOR)
			if err != nil {
				log.Println("Error reading line:", err)
				onStream(types.OnStreamPlanParams{Msg: nil, Err: err})
				body.Close()
				return
			}

			var msg shared.StreamMessage
			err = json.Unmarshal([]byte(s), &msg)
			if err != nil {
				log.Println("Error unmarshalling message:", err)
				onStream(types.OnStreamPlanParams{Msg: nil, Err: err})
				body.Close()
				return
			}

			// log.Println("Received message:", msg)

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

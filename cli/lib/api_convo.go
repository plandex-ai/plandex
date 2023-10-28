package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/plandex/plandex/shared"
)

func (api *API) ConvoSummary(rootId, latestTimestamp string) (*shared.ConversationSummary, error) {
	serverUrl := apiHost + "/convo-summary/" + rootId

	req, err := http.NewRequest("GET", serverUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %s", err)
	}

	if latestTimestamp != "" {
		q := req.URL.Query()
		q.Add("latestTimestamp", latestTimestamp)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to server: %s", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		// Read the error message from the body
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned an error %d: %s", resp.StatusCode,
			string(errorBody))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %s", err)
	}

	var convoSummary shared.ConversationSummary
	err = json.Unmarshal(body, &convoSummary)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response body: %s", err)
	}

	return &convoSummary, nil
}

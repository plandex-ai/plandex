package lib

import (
	"fmt"
	"io"
	"net/http"
)

func (api *API) Abort(proposalId string) error {
	fmt.Println("api aborting proposal", proposalId)

	serverUrl := apiHost + "/abort"

	req, err := http.NewRequest("DELETE", serverUrl, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %s", err)
	}

	q := req.URL.Query()
	q.Add("proposalId", proposalId)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to server: %s", err)
	}

	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		// Read the error message from the body
		errorBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned an error %d: %s", resp.StatusCode,
			string(errorBody))
	}

	return nil
}

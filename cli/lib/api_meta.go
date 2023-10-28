package lib

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/plandex/plandex/shared"
)

func (api *API) FileName(text string) (*shared.FileNameResponse, error) {
	serverUrl := apiHost + "/filename"

	payload := shared.FileNameRequest{
		Text: text,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(serverUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// fmt.Println("FileName response body:")
	// fmt.Println(string(body))

	var fileName shared.FileNameResponse
	err = json.Unmarshal(body, &fileName)
	if err != nil {
		return nil, err
	}

	return &fileName, nil
}

func (api *API) ShortSummary(text string) (*shared.ShortSummaryResponse, error) {
	serverUrl := apiHost + "/summarize"

	payload := shared.ShortSummaryRequest{
		Text: text,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(serverUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// fmt.Println("ShortSummary response body:")
	// fmt.Println(string(body))

	var summarized shared.ShortSummaryResponse
	err = json.Unmarshal(body, &summarized)
	if err != nil {
		return nil, err
	}

	return &summarized, nil
}

package clouding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	ENDPOINT = "https://api.clouding.io"
	VERSION  = "v1"
)

type API struct {
	Endpoint string
	Token    string
}

type ErrorResponse struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
	TraceID  string `json:"traceId,omitempty"`
	// Errors is kept as raw JSON because the API returns field-level validation
	// errors as an object ({"field": ["msg"]}), not as an array. Decoding it into
	// a fixed Go shape made every validation error fail to decode, masking the
	// real message behind "cannot unmarshal object into Go struct field".
	Errors json.RawMessage `json:"errors,omitempty"`
}

// ValidationErrors returns the raw field-level validation errors as a string,
// or an empty string when the response carried none.
func (e ErrorResponse) ValidationErrors() string {
	if len(e.Errors) == 0 {
		return ""
	}
	return string(e.Errors)
}

type option func(*API) error

func NewAPI(token string, options ...option) (*API, error) {
	api := API{
		Endpoint: ENDPOINT,
		Token:    token,
	}

	for _, option := range options {
		err := option(&api)
		if err != nil {
			return nil, err
		}
	}
	return &api, nil
}

func WithEndpoint(endpoint string) option {
	return func(a *API) error {
		a.Endpoint = endpoint
		return nil
	}
}

func (a *API) sendRequest(method string, path string, body []byte) (*http.Response, error) {
	request, err := http.NewRequest(method, fmt.Sprintf("%s/%s/%s", a.Endpoint, VERSION, path), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	// Set headers
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-KEY", a.Token)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

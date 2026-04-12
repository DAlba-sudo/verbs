package verbs

import "net/http"

type Request struct {
}

func (req Request) Data(w http.ResponseWriter, r *http.Request, model map[string]any) (any, error) {
	method := r.Method
	url := r.URL.Path

	var values map[string][]string
	if err := r.ParseForm(); err == nil {
		values = r.Form
	}

	var payload struct {
		Method string
		URL    string
		Values map[string][]string
	}

	payload.Method = method
	payload.URL = url
	payload.Values = values
	return payload, nil
}

func (r Request) Name() string {
	return "Request"
}

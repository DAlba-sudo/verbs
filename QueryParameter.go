package verbs

import (
	"net/http"
	"strings"
)

// The query parameter bridge will aggregate a given key's values and expose them as
// a list of strings in the model map for other bridges to consume.

type queryParameter struct {
	TemplateName string
	key          string
	first        bool
}

type QueryParameterOptions struct {
	// this allows you to override the name the values will take
	// in the template, should you need to access them that way.
	Name  string
	First bool
}

func QueryParameter(key string, opts *QueryParameterOptions) queryParameter {
	name := key
	if opts != nil && opts.Name != "" {
		name = opts.Name
	}

	return queryParameter{
		TemplateName: name,
		key:          key,
		first:        opts.First,
	}
}

func (q queryParameter) Data(w http.ResponseWriter, r *http.Request, model map[string]any) (any, error) {
	items := []string{}
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	for _, values := range r.Form[q.key] {
		for value := range strings.SplitSeq(values, ",") {
			v := strings.TrimSpace(value)
			if v != "" {
				items = append(items, v)
			}
		}
	}

	if q.first && len(items) > 0 {
		return items[0], nil
	} else if q.first {
		return nil, nil
	}

	return items, nil
}

func (q queryParameter) Name() string {
	return q.TemplateName
}

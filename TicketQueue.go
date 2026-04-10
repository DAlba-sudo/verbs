package verbs

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/DAlba-sudo/verb/htmx"
)

var (
	logger = slog.Default()
)

type ticketQ struct {
	// the max number of allowed concurrent requests
	MaxConcurrentRequests int    `json:"size"`
	Delay                 int    `json:"delay"`
	URL                   string `json:"url"`
	Include               string `json:"include"`

	// this is the action that the ticket queue will perform
	// when the ticket is successfully acquired.
	Action func(r *http.Request, m map[string]any) (any, error)

	// the ticket queue to use
	tickets chan struct{}
}

// This function returns a TicketQ, a system that ensures only `size` amount
// of concurrent executions of `Action` occur.
func TicketQ(size int, delay int, url string, include string, action func(r *http.Request, w map[string]any) (any, error)) *ticketQ {
	return &ticketQ{
		MaxConcurrentRequests: size,
		Action:                action,
		Delay:                 delay,
		URL:                   url,
		Include:               include,

		tickets: make(chan struct{}, size),
	}
}

func (t ticketQ) Name() string { return "Ticket" }

func (t ticketQ) Data(w http.ResponseWriter, r *http.Request, model map[string]any) (any, error) {

	select {
	case t.tickets <- struct{}{}:
		// the action will be performed, and the ticket will be
		// consumed.
		defer func() { <-t.tickets }()

		v, err := t.Action(r, model)
		if v == nil || err != nil {
			// since the action failed, we will do nothing for now.
			return nil, err
		}

		return v, nil
	default:
		// the action could not be performed
		// so the default action was taken.

		v, ok := model["Htmx"]
		if !ok || v == nil {
			// in this case, the Htmx verb system is not being
			// used so we just return nil, nil
			logger.Warn("TicketQ failed to retrieve HTMX from Map", "v", v, "ok", ok)
			return nil, nil
		}

		hx, ok := v.(htmx.Htmx)
		if !ok {
			// there was a failure during the casting, so
			// we will return something basic.
			logger.Warn("TicketQ failed to cast HTMX from Map", "v", v)
			return nil, nil
		}

		hx.Trigger(fmt.Sprintf("load delay:%ds", t.Delay)).
			GET(t.URL).
			Swap("outerHTML")

		if t.Include != "" {
			hx.Include(t.Include)
		}
		model["Htmx"] = hx
	}

	return nil, nil
}

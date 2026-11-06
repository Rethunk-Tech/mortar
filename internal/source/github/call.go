package github

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
	ghauth "github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// call is one request to GitHub. A 403 or 429 comes back as the busy error, recorded in *Refused while GitHub's
// limit is in force when Refused is set; any other status that is not 200 is an *httpkit.StatusError.
type call struct {
	Method  string
	URL     string
	Version string
	Accept  string
	// Body, when set, is sent as JSON.
	Body []byte
	// Auth sends the user's gh login token, for api.github.com.
	Auth    bool
	Refused **source.BusyError
}

// reply is what survives of a response once its body is read.
type reply struct {
	Status int
	Header http.Header
}

// do sends c and returns the body of a 200 answer with its status and headers, which are set whenever the server
// answered.
func (d *Driver) do(ctx context.Context, c call) ([]byte, reply, error) {
	ctx, cancel := context.WithTimeout(ctx, source.RequestTimeout)
	defer cancel()
	var payload io.Reader
	if c.Body != nil {
		payload = bytes.NewReader(c.Body)
	}
	req, err := http.NewRequestWithContext(ctx, c.Method, c.URL, payload)
	if err != nil {
		return nil, reply{}, err
	}
	req.Header.Set("User-Agent", appversion.UserAgent(c.Version))
	if c.Accept != "" {
		req.Header.Set("Accept", c.Accept)
	}
	if c.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Auth {
		ghauth.DefaultAuth.Apply(req)
	}
	var got reply
	body, err := httpkit.DoCapped(d.HTTP, req, source.MaxBody, func(r *http.Response) error {
		got = reply{r.StatusCode, r.Header}
		if r.StatusCode != http.StatusForbidden && r.StatusCode != http.StatusTooManyRequests {
			return nil
		}
		if c.Refused != nil {
			return d.refuse(c.Refused, r)
		}
		return source.Busy("GitHub", r)
	})
	return body, got, err
}

// answered words an unexpected status as "<service> answered <status>" and leaves any other error alone.
func answered(service string, err error) error {
	if se, ok := errors.AsType[*httpkit.StatusError](err); ok {
		return fmt.Errorf("%s answered %s", service, se.Status)
	}
	return err
}

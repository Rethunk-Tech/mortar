package nexus

import (
	"net/http"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/mortar/internal/netstate"
)

// ObservedHTTP is the http client for go-nexusmods: every round trip it makes, a collection mirror's included, tells
// netstate whether Nexus was reached. base carries the User-Agent.
func ObservedHTTP(base http.RoundTripper) *http.Client {
	return &http.Client{Transport: &httpkit.Transport{Base: base, OnResult: func(_ *http.Request, _ *http.Response, err error) {
		netstate.RecordTransport("nexus", err)
	}}}
}

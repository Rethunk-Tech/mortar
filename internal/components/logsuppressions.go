package components

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/meta"
)

// LogSuppressionsURL is where each game's signed list of log lines hidden by default is published
// (Rethunk-Tech/mortar-log-suppressions, the rolling release "lists"): <game>.json and <game>.json.sig.
const LogSuppressionsURL = "https://github.com/Rethunk-Tech/mortar-log-suppressions/releases/download/lists/"

// LogSuppression is a source of game log lines hidden unless the player shows it again, as settings.HiddenLogSource
// names one.
type LogSuppression struct {
	Source string `json:"source"`
	Text   string `json:"text,omitempty"`
	Object string `json:"object,omitempty"`
	Label  string `json:"label"`
}

// LogSuppressionList is one game's signed list of log lines hidden by default.
type LogSuppressionList struct {
	Game    string           `json:"game"`
	Serial  uint64           `json:"serial"`
	Entries []LogSuppression `json:"entries"`
}

// DecodeLogSuppressions parses and checks a list for game.
func DecodeLogSuppressions(body []byte, game string) (LogSuppressionList, error) {
	var l LogSuppressionList
	if err := json.Unmarshal(body, &l); err != nil {
		return LogSuppressionList{}, err
	}
	if l.Game != game || l.Serial == 0 {
		return LogSuppressionList{}, fmt.Errorf("log suppression list for %q has game %q, serial %d", game, l.Game, l.Serial)
	}
	for i, e := range l.Entries {
		if e.Source == "" || e.Label == "" {
			return LogSuppressionList{}, fmt.Errorf("log suppression %d needs a source and a label", i)
		}
	}
	return l, nil
}

var logSuppressions = signedList[LogSuppressionList]{
	prefix: meta.LogSuppressionsPrefix,
	url:    LogSuppressionsURL,
	decode: func(body []byte, game string) (LogSuppressionList, uint64, error) {
		l, err := DecodeLogSuppressions(body, game)
		return l, l.Serial, err
	},
}

var suppressionLists listMemo[LogSuppressionList]

// LogSuppressionsFor is game's list of log lines hidden by default, empty when none is published or none could be
// fetched yet.
func LogSuppressionsFor(ctx context.Context, game string) LogSuppressionList {
	return suppressionLists.get(ctx, logSuppressions, game)
}

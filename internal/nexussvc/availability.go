package nexussvc

import (
	"strings"
	"time"
)

// PageMark is how a Nexus mod page that is no longer published is shown.
// An empty Kind means the page is usable; running the installed mod is never blocked.
type PageMark struct {
	Kind string `json:"kind"` // "removed" or "hidden"
	Date string `json:"date"` // YYYY-MM-DD when known
}

func dateDay(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02")
}

// Mark classifies this page: hidden when available is false, else removed when status is not published.
func (d Details) Mark() PageMark {
	return PageAvailability(d.Page.Status, d.Page.Available, dateDay(d.Page.Updated), dateDay(d.Page.Created))
}

// PageAvailability classifies cached or fetched mod-page details.
// status other than "published", or available false, means updates cannot be downloaded.
func PageAvailability(status string, available bool, updatedTime, createdTime string) PageMark {
	date := strings.TrimSpace(updatedTime)
	if date == "" {
		date = strings.TrimSpace(createdTime)
	}
	if i := strings.IndexByte(date, 'T'); i > 0 {
		date = date[:i]
	}
	if !available {
		return PageMark{Kind: "hidden", Date: date}
	}
	st := strings.TrimSpace(strings.ToLower(status))
	if st != "" && st != "published" {
		return PageMark{Kind: "removed", Date: date}
	}
	return PageMark{}
}

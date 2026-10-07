package profile

import "testing"

func TestHistoryItemDetailsReadAsSentences(t *testing.T) {
	items := historyItems(HistoryDiff{
		Added:   []DiffMod{{Name: "Alpacas"}},
		Removed: []DiffMod{{Name: "Bees"}},
	})
	if len(items) != 2 || items[0].Detail != "Added Alpacas" || items[1].Detail != "Removed Bees" {
		t.Fatalf("items = %+v", items)
	}
}

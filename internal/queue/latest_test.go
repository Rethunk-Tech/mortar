package queue

import (
	"testing"

	"github.com/Rethunk-AI/mortar/internal/nexus"
)

func TestNewestUpdateFollowsTheChain(t *testing.T) {
	files := []nexus.File{
		{FileID: 1, Version: "1.0.0", ReplacedBy: 2},
		{FileID: 2, Version: "1.1.0", ReplacedBy: 3},
		{FileID: 3, Version: "1.2.0"},
		{FileID: 9, Version: "0.1.0", ReplacedBy: 9},  // a file that names itself must not loop
		{FileID: 5, Version: "2.0.0", ReplacedBy: 42}, // a successor that was deleted keeps the last listed file
	}
	cases := map[int]int{1: 3, 2: 3, 3: 3, 9: 9, 5: 5}
	for from, want := range cases {
		if got := newestUpdate(files, fileByID(files, from)).FileID; got != want {
			t.Errorf("from %d: got %d, want %d", from, got, want)
		}
	}
}

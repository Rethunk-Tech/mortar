package sharesvc

import (
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

func TestPickModPageFile(t *testing.T) {
	old, now := time.Unix(100, 0), time.Unix(200, 0)
	if f := pickModPageFile([]nexus.File{{FileID: 1, Category: "MAIN"}, {FileID: 2, IsPrimary: true}}); f == nil || f.FileID != 2 {
		t.Fatalf("primary: %+v", f)
	}
	if f := pickModPageFile([]nexus.File{{FileID: 1, Category: "MAIN", Uploaded: old}, {FileID: 3, Category: "MAIN", Uploaded: now}, {FileID: 4, Category: "OPTIONAL", Uploaded: now}}); f == nil || f.FileID != 3 {
		t.Fatalf("newest main: %+v", f)
	}
	if f := pickModPageFile([]nexus.File{{FileID: 4, Category: "OPTIONAL"}}); f != nil {
		t.Fatalf("optional only: %+v", f)
	}
}

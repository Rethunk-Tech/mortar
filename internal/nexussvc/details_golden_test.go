package nexussvc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	nexusmods "github.com/Rethunk-Tech/go-nexusmods"
	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

// detailsGolden is a details-v4 cache file as this build writes it. Players' caches hold files of this shape, so a
// field or tag change that breaks reading it back fails here rather than serving empty details.
const detailsGolden = `{"fetched":"2026-10-30T01:00:00Z","value":{"page":{"modId":541,"name":"Mod","summary":"S","description":"D","pictureUrl":"https://staticdelivery.nexusmods.com/p.png","version":"1.2.0","author":"A","uploadedBy":"U","uploaderUrl":"https://www.nexusmods.com/users/1","categoryId":7,"endorsements":3,"downloads":40,"uniqueDownloads":20,"created":"2026-01-02T03:04:05Z","updated":"2026-02-03T04:05:06Z","adult":true,"status":"published","available":true,"endorsement":"Endorsed","requirements":[{"modId":2400,"name":"SMAPI","url":"https://www.nexusmods.com/stardewvalley/mods/2400","external":false,"notes":"n"}]},"category":"Gameplay","files":[{"fileId":9,"fileName":"Mod-541-1-2-0.zip","name":"Main","description":"F","version":"1.2.0","modVersion":"1.2.0","category":"MAIN","sizeKb":12,"isPrimary":true,"uploaded":"2026-02-03T04:05:06Z","md5":"0123456789abcdef0123456789abcdef","replacedBy":10}],"changelogs":[{"version":"1.2.0","date":"2026-02-03","notes":["fixed"],"body":"b"}],"partial":true}}`

func detailsGoldenValue() diskcache.Entry[Details] {
	at := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			panic(err)
		}
		return v
	}
	return diskcache.Entry[Details]{
		Fetched: at("2026-10-30T01:00:00Z"),
		Value: Details{
			Page: nexusmods.Page{
				ModID: 541, Name: "Mod", Summary: "S", Description: "D",
				PictureURL: "https://staticdelivery.nexusmods.com/p.png", Version: "1.2.0", Author: "A",
				UploadedBy: "U", UploaderURL: "https://www.nexusmods.com/users/1", CategoryID: 7, Endorsements: 3,
				Downloads: 40, UniqueDownloads: 20, Created: at("2026-01-02T03:04:05Z"), Updated: at("2026-02-03T04:05:06Z"),
				Adult: true, Status: "published", Available: true, Endorsement: "Endorsed",
				Requirements: []nexusmods.Requirement{{
					ModID: 2400, Name: "SMAPI", URL: "https://www.nexusmods.com/stardewvalley/mods/2400", Notes: "n",
				}},
			},
			Category: "Gameplay",
			Files: []nexusmods.File{{
				FileID: 9, FileName: "Mod-541-1-2-0.zip", Name: "Main", Description: "F", Version: "1.2.0",
				ModVersion: "1.2.0", Category: "MAIN", SizeKB: 12, IsPrimary: true, Uploaded: at("2026-02-03T04:05:06Z"),
				MD5: "0123456789abcdef0123456789abcdef", ReplacedBy: 10,
			}},
			Changelogs: []nexus.Changelog{{Version: "1.2.0", Date: "2026-02-03", Notes: []string{"fixed"}, Body: "b"}},
			Partial:    true,
		},
	}
}

func TestDetailsCacheFileRoundTrips(t *testing.T) {
	t.Parallel()
	want := detailsGoldenValue()
	got, err := json.Marshal(want)
	if err != nil || string(got) != detailsGolden {
		t.Fatalf("written cache file changed shape (%v):\n%s", err, got)
	}
	dir := t.TempDir()
	name := DetailsName("stardewvalley", 541)
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(detailsGolden), 0o600); err != nil {
		t.Fatal(err)
	}
	read, ok := diskcache.Peek[Details](&diskcache.Cache{Dir: dir}, name)
	if !ok || !reflect.DeepEqual(read.Value, want.Value) {
		t.Fatalf("cached details read back as %+v (found %v)", read.Value, ok)
	}
}

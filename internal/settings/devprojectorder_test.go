package settings

import (
	"slices"
	"testing"
)

func TestSetDevProjectOrder(t *testing.T) {
	cases := []struct {
		name    string
		order   []string
		want    []string
		wantErr bool
	}{
		{"reverse the lot", []string{"d", "c", "b", "a"}, []string{"d", "c", "b", "a"}, false},
		{"subset keeps the other slots", []string{"c", "a"}, []string{"c", "b", "a", "d"}, false},
		{"unknown dirs are ignored", []string{"x", "b", "a"}, []string{"b", "a", "c", "d"}, false},
		{"empty changes nothing", nil, []string{"a", "b", "c", "d"}, false},
		{"duplicate is an error", []string{"a", "b", "a"}, []string{"a", "b", "c", "d"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store, _ := open(t)
			svc := NewService(store)
			for _, d := range []string{"a", "b", "c", "d"} {
				if err := svc.AddDevProject("g", DevProject{Dir: d, Name: d, Output: d}); err != nil {
					t.Fatal(err)
				}
			}
			err := svc.SetDevProjectOrder("g", c.order)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, want error %v", err, c.wantErr)
			}
			var got []string
			for _, p := range store.Get().GamePrefs("g").DevProjects {
				got = append(got, p.Dir)
			}
			if !slices.Equal(got, c.want) {
				t.Fatalf("order = %v, want %v", got, c.want)
			}
		})
	}
}

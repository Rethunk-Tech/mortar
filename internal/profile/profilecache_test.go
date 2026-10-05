package profile

import (
	"reflect"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// A profile handed out from the cache shares nothing a caller can change with the cached one or another copy.
func TestCachedProfileCopiesShareNothing(t *testing.T) {
	src := Source{Kind: KindNexus, ModID: 1}
	p := Profile{
		Entries: []Entry{{
			Key: "a", PreviousSource: &src, Mods: []Component{{ID: "smapi:A", Needs: []mod.ID{"smapi:B"}}},
			Disabled: []mod.ID{"smapi:A"}, Tags: []string{"t"}, Fomod: map[string]map[string][]string{"s": {"g": {"o"}}},
		}},
		Groups: []Group{{Name: "G", Keys: []string{"a"}}}, Collection: &CollectionRef{Slug: "c"},
		LaunchPresets: []LaunchPreset{{Name: "P"}}, Overrides: map[string]string{"k": "v"},
	}
	c := cloneProfile(p)
	if !reflect.DeepEqual(c, p) {
		t.Fatalf("copy differs: %+v", c)
	}
	assertUnshared(t, "Profile", reflect.ValueOf(p), reflect.ValueOf(c))
}

func assertUnshared(t *testing.T, path string, a, b reflect.Value) {
	t.Helper()
	k := a.Kind()
	if k == reflect.Pointer || k == reflect.Slice || k == reflect.Map {
		if a.IsNil() {
			return
		}
		if a.Pointer() == b.Pointer() {
			t.Errorf("%s is shared", path)
		}
	}
	switch k {
	case reflect.Pointer:
		assertUnshared(t, path, a.Elem(), b.Elem())
	case reflect.Slice:
		for i := range a.Len() {
			assertUnshared(t, path+"[]", a.Index(i), b.Index(i))
		}
	case reflect.Map:
		for it := a.MapRange(); it.Next(); {
			assertUnshared(t, path+"{}", it.Value(), b.MapIndex(it.Key()))
		}
	case reflect.Struct:
		for i := range a.NumField() {
			if a.Type().Field(i).IsExported() {
				assertUnshared(t, path+"."+a.Type().Field(i).Name, a.Field(i), b.Field(i))
			}
		}
	case reflect.Array, reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Func, reflect.Interface, reflect.String, reflect.UnsafePointer:
	}
}

// A read after a write sees the write, not the profile cached before it.
func TestProfileReadAfterWriteSeesTheWrite(t *testing.T) {
	e := newEnv(t)
	p := mustCreate(t, e, "Farm")
	if got, err := e.read("stardew", p.ID); err != nil || got.Name != "Farm" {
		t.Fatalf("first read: %q %v", got.Name, err)
	}
	if _, err := e.Rename("stardew", p.ID, "Ranch"); err != nil {
		t.Fatal(err)
	}
	if again, err := e.read("stardew", p.ID); err != nil || again.Name != "Ranch" {
		t.Fatalf("read after a write: %q %v", again.Name, err)
	}
}

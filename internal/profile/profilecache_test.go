package profile

import (
	"reflect"
	"testing"
	"time"
)

// populate fills every exported field reachable from v with a non-zero value: slices, maps and pointers are non-nil
// with one element each, so a field Clone forgets to copy is shared and fails the check below.
func populate(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
		populate(v.Elem())
	case reflect.Slice:
		v.Set(reflect.MakeSlice(v.Type(), 1, 1))
		populate(v.Index(0))
	case reflect.Map:
		v.Set(reflect.MakeMap(v.Type()))
		k, e := reflect.New(v.Type().Key()).Elem(), reflect.New(v.Type().Elem()).Elem()
		populate(k)
		populate(e)
		v.SetMapIndex(k, e)
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				populate(v.Field(i))
			}
		}
	case reflect.String:
		v.SetString("x")
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(1)
	case reflect.Array, reflect.Invalid, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Func, reflect.Interface,
		reflect.UnsafePointer:
	}
}

// A profile handed out from the cache shares nothing a caller can change with the cached one or another copy, whatever
// fields Profile has.
func TestProfileCloneSharesNothing(t *testing.T) {
	var p Profile
	populate(reflect.ValueOf(&p).Elem())
	p.Created, p.Updated = time.Unix(1, 0).UTC(), time.Unix(2, 0).UTC()
	p.Entries[0].Added = time.Unix(3, 0).UTC()
	c := p.Clone()
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

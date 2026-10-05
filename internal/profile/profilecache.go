package profile

import (
	"os"
	"reflect"
	"sync"
)

// parsedProfiles holds each profile.json decoded, by path, while the file is still the one it was decoded from: every
// write replaces the file through a rename, so a changed identity, size or modification time means it was rewritten.
// A start decodes each profile dozens of times (every List), and the file only changes when Mortar writes it.
var parsedProfiles sync.Map

type parsedProfile struct {
	file os.FileInfo
	p    Profile
}

func cachedProfile(path string, fi os.FileInfo) (Profile, bool) {
	c, ok := parsedProfiles.Load(path)
	if !ok {
		return Profile{}, false
	}
	pp, ok := c.(parsedProfile)
	if !ok || !os.SameFile(fi, pp.file) || !fi.ModTime().Equal(pp.file.ModTime()) || fi.Size() != pp.file.Size() {
		return Profile{}, false
	}
	return cloneProfile(pp.p), true
}

func rememberProfile(path string, fi os.FileInfo, p Profile) {
	parsedProfiles.Store(path, parsedProfile{fi, cloneProfile(p)})
}

func forgetProfile(path string) { parsedProfiles.Delete(path) }

// cloneProfile copies p with no slice, map or pointer shared, so a caller changing its copy never reaches the cache.
func cloneProfile(p Profile) Profile {
	unshare(reflect.ValueOf(&p).Elem())
	return p
}

// unshare replaces every slice, map and pointer reachable from v through exported fields with a copy of its own.
// Unexported fields are left: decoding never sets them.
func unshare(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() || !v.CanSet() {
			return
		}
		n := reflect.New(v.Type().Elem())
		n.Elem().Set(v.Elem())
		unshare(n.Elem())
		v.Set(n)
	case reflect.Slice:
		if v.IsNil() || !v.CanSet() {
			return
		}
		n := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		reflect.Copy(n, v)
		for i := range n.Len() {
			unshare(n.Index(i))
		}
		v.Set(n)
	case reflect.Map:
		if v.IsNil() || !v.CanSet() {
			return
		}
		n := reflect.MakeMapWithSize(v.Type(), v.Len())
		for it := v.MapRange(); it.Next(); {
			val := reflect.New(v.Type().Elem()).Elem()
			val.Set(it.Value())
			unshare(val)
			n.SetMapIndex(it.Key(), val)
		}
		v.Set(n)
	case reflect.Struct:
		for _, f := range v.Fields() {
			unshare(f)
		}
	case reflect.Array:
		for i := range v.Len() {
			unshare(v.Index(i))
		}
	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Func, reflect.Interface, reflect.String, reflect.UnsafePointer:
	}
}

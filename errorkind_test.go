package main

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

type kindProbe struct{}

func (*kindProbe) Fail() error { return usererr.Wrap(usererr.Busy, errors.New("running")) }

// A service registered without its own MarshalError must still hand the window the app-wide usererr kind.
func TestBoundCallErrorCarriesUsererrKind(t *testing.T) {
	_ = application.New(application.Options{})
	bindings := application.NewBindings(usererr.Marshal, nil)
	if err := bindings.Add(application.NewService(&kindProbe{})); err != nil {
		t.Fatal(err)
	}
	_, err := bindings.Get(&application.CallOptions{MethodName: reflect.TypeFor[kindProbe]().PkgPath() + ".kindProbe.Fail"}).Call(context.Background(), nil)
	var cerr *application.CallError
	if !errors.As(err, &cerr) {
		t.Fatalf("err = %#v", err)
	}
	if raw, ok := cerr.Cause.(json.RawMessage); !ok || string(raw) != `{"kind":"busy"}` {
		t.Fatalf("cause = %#v", cerr.Cause)
	}
}

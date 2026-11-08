package vsmoddb

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rethunk-Tech/go-vintagestory/moddb"
)

func TestReportTargetIsTheModPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"mod":{"modid":5,"assetid":50,"urlalias":"fine"}}`))
	}))
	t.Cleanup(srv.Close)
	got, err := Driver{HTTP: srv.Client(), URL: srv.URL}.ReportTarget(t.Context(), "", "5-50", "")
	if err != nil || got.URL != moddb.SiteURL+"/fine" || got.Repo != "" {
		t.Fatalf("target %+v, %v", got, err)
	}
}

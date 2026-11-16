package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIsolatedBrowserSearch(t *testing.T) {
	if os.Getenv("TEACHING_BROWSER_QA") != "1" {
		t.Skip("requires installed Chrome/Edge; deterministic browser QA opt-in")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" {
			t.Error("unexpected shared cookie")
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><script>const h=document.createElement('h2'),a=document.createElement('a');a.href='https://example.com/evidence';a.textContent='Browser evidence';h.append(a);document.body.append(h)</script></body></html>`))
	}))
	defer srv.Close()
	dir := t.TempDir()
	a := &App{profiles: &ProfileStore{path: filepath.Join(dir, "profiles.json")}}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	rows, e := a.browserSearch(ctx, searchEngine{ID: "bing_cn", URL: srv.URL}, "example", false)
	if e != nil || len(rows) != 1 || !strings.Contains(rows[0].Engine, "浏览器") {
		t.Fatal(rows, e)
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 0 {
		t.Fatal("isolated browser cache not cleaned")
	}
}

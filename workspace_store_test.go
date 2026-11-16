package main

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceRecoveryAndConcurrentWriters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.json")
	a := &App{profiles: &ProfileStore{path: path}}
	call := func(app *App, body string) (int, map[string]any) {
		w := httptest.NewRecorder()
		app.workspaceAPI(w, httptest.NewRequest("POST", "/api/workspaces", strings.NewReader(body)))
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}
	raw := `{"action":"save","id":"test-window-one","title":"课程甲","revision":0,"state":{"version":1,"sessions":{"report":{"jobState":"running","completed":["p1"]}}}}`
	if code, _ := call(a, raw); code != 200 {
		t.Fatal(code)
	}
	// A new application instance must load the exact committed checkpoint.
	b := &App{profiles: &ProfileStore{path: path}}
	code, x := call(b, `{"action":"load","id":"test-window-one"}`)
	if code != 200 || x["revision"] != float64(1) {
		t.Fatalf("%d %v", code, x)
	}
	if code, _ = call(b, raw); code != 409 {
		t.Fatalf("stale writer overwrote record: %d", code)
	}
	if code, _ = call(b, `{"action":"save","id":"test-window-one","revision":1,"state":{"version":99}}`); code != 400 {
		t.Fatal("bad schema accepted")
	}
	code, x = call(a, `{"action":"load","id":"test-window-one"}`)
	state, _ := json.Marshal(x["state"])
	if !strings.Contains(string(state), "p1") {
		t.Fatal("checkpoint lost")
	}
	if code, _ = call(a, `{"action":"load","id":"../escape"}`); code != 400 {
		t.Fatal("unsafe ID accepted")
	}
	if code, _ = call(a, `{"action":"save","id":"second-window","revision":0,"state":{"version":1}}`); code != 200 {
		t.Fatal("independent session failed")
	}
	_, x = call(a, `{"action":"list"}`)
	if len(x["sessions"].([]any)) != 2 {
		t.Fatal(x)
	}
	g := windowGeometry{X: 500, Y: 200, Width: 1200, Height: 800}
	if e := a.saveGeometry("workbench", g); e != nil {
		t.Fatal(e)
	}
	if got, ok := b.loadGeometry("workbench"); !ok || got != g {
		t.Fatal(got)
	}
}
func TestWindowGeometryRemovedMonitor(t *testing.T) {
	g := clampGeometry(windowGeometry{X: 5000, Y: -2000, Width: 3000, Height: 2000}, 0, 0, 1280, 720)
	if g.X < 0 || g.Y < 0 || g.X+g.Width > 1280 || g.Y+g.Height > 720 {
		t.Fatal(g)
	}
	g = clampGeometry(windowGeometry{X: -1800, Y: 30, Width: 1200, Height: 700}, -1920, 0, 1920, 1080)
	if g.X != -1800 {
		t.Fatal("secondary monitor position lost", g)
	}
}

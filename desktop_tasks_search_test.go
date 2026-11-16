package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTeacherTravelTimingAndBoundaries(t *testing.T) {
	for _, fps := range []int{30, 60, 120} {
		m := teacherRoam{}
		now := time.Unix(1, 0)
		x, y := 500, 500
		for i := 0; i <= fps; i++ {
			x, y, _ = m.step(now.Add(time.Duration(float64(i)/float64(fps)*float64(time.Second))), x, y, 0, 0, 1200, 900, 120, true)
		}
		if x > 405 || x < 380 || y > 465 || y < 440 {
			t.Fatalf("fps=%d x=%d y=%d", fps, x, y)
		}
		px, py := x, y
		x, y, _ = m.step(now.Add(10*time.Second), x, y, 0, 0, 1200, 900, 120, false)
		if x != px || y != py {
			t.Fatal("paused movement")
		}
		for i := 0; i < 2000; i++ {
			x, y, _ = m.step(now.Add(time.Duration(i)*time.Second/60), x, y, 20, 30, 210, 240, 220, true)
			if x < 20 || x > 210 || y < 30 || y > 240 {
				t.Fatal("escaped bounds")
			}
		}
	}
}
func TestModuleTasksPersistAndConflict(t *testing.T) {
	a := &App{profiles: &ProfileStore{path: filepath.Join(t.TempDir(), "profiles.json")}}
	call := func(body string) (int, map[string]any) {
		w := httptest.NewRecorder()
		a.moduleTasks(w, httptest.NewRequest("POST", "/api/module-tasks", strings.NewReader(body)))
		var x map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &x)
		return w.Code, x
	}
	code, _ := call(`{"action":"save","module":"report","id":"task-test-1","title":"报告一","state":{"version":1,"session":{"conversation":[],"reviewResults":[]}}}`)
	if code != 200 {
		t.Fatal(code)
	}
	code, x := call(`{"action":"list","module":"report"}`)
	if code != 200 || len(x["tasks"].([]any)) != 1 {
		t.Fatal(x)
	}
	_, x = call(`{"action":"list","module":"presentation"}`)
	if len(x["tasks"].([]any)) != 0 {
		t.Fatal("module leak")
	}
	code, _ = call(`{"action":"save","module":"report","id":"task-test-1","title":"overwrite","state":{"version":1,"session":{}}}`)
	if code != 409 {
		t.Fatal("stale overwrite", code)
	}
}
func TestWorkBuddyDetection(t *testing.T) {
	db, e := sql.Open(sqliteDriver, filepath.Join(t.TempDir(), "wb.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	ctx := context.Background()
	if isWorkBuddyLibrary(ctx, db) {
		t.Fatal("false positive")
	}
	_, e = db.Exec("CREATE TABLE meta(key TEXT,value TEXT);INSERT INTO meta VALUES('wb_format','workbuddy')")
	if e != nil {
		t.Fatal(e)
	}
	if !isWorkBuddyLibrary(ctx, db) {
		t.Fatal("marker ignored")
	}
	_, _ = db.Exec("DROP TABLE meta;CREATE TABLE wb_acceptance(check_id TEXT);INSERT INTO wb_acceptance VALUES('D1')")
	if !isWorkBuddyLibrary(ctx, db) {
		t.Fatal("acceptance marker ignored")
	}
}
func TestWebSearchAdaptersAndValidation(t *testing.T) {
	s := searchDefaults()
	if e := validateSearch(s); e != nil {
		t.Fatal(e)
	}
	s.Engines[0].URL = "https://user:secret@example.com"
	if validateSearch(s) == nil {
		t.Fatal("credentials in URL")
	}
	raw := `{"Response":{"WebPages":[{"Title":"课程证据","Url":"https://example.com/evidence","Summary":"真实摘要"}]}}`
	var v any
	_ = json.Unmarshal([]byte(raw), &v)
	if r := searchJSONResults(v, "yuanbao"); len(r) != 1 || r[0].Snippet != "真实摘要" {
		t.Fatal(r)
	}
	r := searchHTMLResults(`<h2><a href="https://example.com/a">课程 <b>教学</b></a></h2>`, "bing_cn")
	if len(r) != 1 || r[0].Title != "课程 教学" {
		t.Fatal(r)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-KEY") != "test-key" {
			t.Error("missing service key")
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["q"] != "教学" {
			t.Error(body)
		}
		_, _ = w.Write([]byte(`{"organic":[{"title":"教学","link":"https://example.com/","snippet":"依据"}]}`))
	}))
	defer server.Close()
	r, e := searchHTTP(context.Background(), searchEngine{ID: "serper", URL: server.URL, NeedsKey: true}, "教学", "test-key")
	if e != nil || len(r) != 1 {
		t.Fatal(r, e)
	}
}

func TestDesktopDragBridge(t *testing.T) {
	desktopState.Lock()
	old := desktopState.control
	var phases []string
	desktopState.control = func(action, raw string) error {
		if action != "drag" {
			t.Fatalf("unexpected action %s", action)
		}
		var d struct {
			Phase string `json:"phase"`
			X, Y  int
		}
		if e := json.Unmarshal([]byte(raw), &d); e != nil {
			t.Fatal(e)
		}
		phases = append(phases, d.Phase)
		return nil
	}
	desktopState.Unlock()
	defer func() {
		desktopState.Lock()
		desktopState.control = old
		desktopState.dragUntil = time.Time{}
		desktopState.Unlock()
	}()
	a := &App{}
	for _, body := range []string{`{"action":"drag","phase":"begin","x":100,"y":200}`, `{"action":"drag","phase":"move","x":180,"y":150}`, `{"action":"drag","phase":"end","x":180,"y":150}`} {
		w := httptest.NewRecorder()
		a.desktop(w, httptest.NewRequest("POST", "/api/desktop", strings.NewReader(body)))
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
	}
	if strings.Join(phases, ",") != "begin,move,end" {
		t.Fatal(phases)
	}
	w := httptest.NewRecorder()
	a.desktop(w, httptest.NewRequest("POST", "/api/desktop", strings.NewReader(`{"action":"drag","phase":"move","x":200000,"y":0}`)))
	if w.Code != 400 {
		t.Fatal("out-of-range coordinate accepted")
	}
	if !desktopState.dragUntil.IsZero() {
		t.Fatal("drag remained active")
	}
}

package main

import (
	"encoding/json"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEndpoint(t *testing.T) {
	for _, s := range []string{"https://user:pass@example.com", "https://example.com/v1?key=x", "file:///x"} {
		if _, e := endpoint(s); e == nil {
			t.Fatalf("accepted unsafe endpoint %s", s)
		}
	}
	for _, s := range []string{"http://example.com/v1", "http://10.0.0.3:8888/v1", "http://[::1]:8888/v1", "https://example.com/v1", "http://127.0.0.1:1234/v1"} {
		if _, e := endpoint(s); e != nil {
			t.Fatal(e)
		}
	}
}
func TestAuthAndChat(t *testing.T) {
	upstreamCalls := 0
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls++
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Error("bad upstream routing")
		}
		var p struct {
			Messages []Message `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&p)
		if len(p.Messages) < 3 || !strings.Contains(p.Messages[1].Content, "R12") {
			t.Error("evidence missing")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"【页】测试证据 [R12]"}}]}`))
	}))
	defer mock.Close()
	a := &App{origin: "http://127.0.0.1:9999", token: "session-test", client: &http.Client{Timeout: time.Second}, cfg: Config{Base: mock.URL + "/v1", Key: "test-secret", Model: "mock"}}
	send := func(path, token, origin, body string) int {
		r := httptest.NewRequest("POST", a.origin+path, strings.NewReader(body))
		r.Header.Set("X-Session-Token", token)
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		a.handler().ServeHTTP(w, r)
		return w.Code
	}
	if c := send("/api/chat", "wrong", "", `{}`); c != 403 {
		t.Fatal(c)
	}
	if c := send("/api/chat", a.token, "https://evil.example", `{}`); c != 403 {
		t.Fatal(c)
	}
	if c := send("/api/chat", a.token, "", `{"question":"测试","evidence":"R12 页面证据"}`); c != 200 {
		t.Fatal(c)
	}
	if upstreamCalls != 1 {
		t.Fatal(upstreamCalls)
	}
	if c := send("/api/clone", a.token, "", `{}`); c != 404 {
		t.Fatal(c)
	}
}
func TestTeacherProfileFollowsDesktopSelection(t *testing.T) {
	a := &App{origin: "http://example.com"}
	desktopState.Lock()
	old := desktopState.template
	desktopState.Unlock()
	defer func() { desktopState.Lock(); desktopState.template = old; desktopState.Unlock() }()
	for _, item := range []struct{ template, persona string }{{"female", "original"}, {"male", "male"}} {
		desktopState.Lock()
		desktopState.template = item.template
		desktopState.Unlock()
		w := httptest.NewRecorder()
		a.handler().ServeHTTP(w, httptest.NewRequest("GET", "http://example.com/profile.json", nil))
		var got map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got["persona"] != item.persona {
			t.Fatalf("profile %s: %s", item.template, w.Body.String())
		}
	}
}

func TestEmbeddedModulesIgnoreHostMimeAssociations(t *testing.T) {
	old := mime.TypeByExtension(".mjs")
	if err := mime.AddExtensionType(".mjs", "text/plain"); err != nil {
		t.Fatal(err)
	}
	defer mime.AddExtensionType(".mjs", old)
	a := &App{origin: "http://127.0.0.1:9999"}
	for _, path := range []string{"/app.mjs", "/pet.mjs", "/teacher-animation.mjs", "/walk-calibration.mjs", "/lib/pdf.mjs"} {
		w := httptest.NewRecorder()
		a.handler().ServeHTTP(w, httptest.NewRequest("GET", a.origin+path, nil))
		if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/javascript") {
			t.Fatalf("%s: status %d MIME %q", path, w.Code, w.Header().Get("Content-Type"))
		}
	}
}

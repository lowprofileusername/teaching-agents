package main

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestWindowBranchPrivateOneTime(t *testing.T) {
	desktopState.Lock()
	old := desktopState.control
	desktopState.control = nil
	desktopState.branches = nil
	desktopState.Unlock()
	defer func() {
		desktopState.Lock()
		desktopState.control = old
		desktopState.branches = nil
		desktopState.Unlock()
	}()
	a := &App{origin: "http://127.0.0.1:9999", token: "test"}
	send := func(body, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", a.origin+"/api/windows", strings.NewReader(body))
		r.Header.Set("X-Session-Token", token)
		w := httptest.NewRecorder()
		a.handler().ServeHTTP(w, r)
		return w
	}
	if w := send(`{"action":"open"}`, "wrong"); w.Code != 403 {
		t.Fatal(w.Code)
	}
	w := send(`{"action":"open","snapshot":{"version":1,"turns":[{"text":"私有草稿"}]}}`, "test")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var x struct {
		URL string `json:"url"`
	}
	json.Unmarshal(w.Body.Bytes(), &x)
	u, _ := url.Parse(x.URL)
	id := u.Query().Get("branch")
	if len(id) != 32 {
		t.Fatal(x.URL)
	}
	body := `{"action":"consume","id":"` + id + `"}`
	w = send(body, "test")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "私有草稿") {
		t.Fatal(w.Code, w.Body.String())
	}
	if send(body, "test").Code != 404 {
		t.Fatal("branch reused")
	}
	desktopState.Lock()
	desktopState.branches["expired"] = desktopBranch{Data: json.RawMessage(`{"version":1}`), At: time.Now().Add(-2 * time.Minute)}
	desktopState.Unlock()
	if send(`{"action":"consume","id":"expired"}`, "test").Code != 404 {
		t.Fatal("expired branch read")
	}
	if send(`{"action":"open","snapshot":{"version":2}}`, "test").Code != 400 {
		t.Fatal("bad version")
	}
	if send(`{"action":"open","snapshot":{"version":1,"text":"`+strings.Repeat("a", 2<<20)+`"}}`, "test").Code != 400 {
		t.Fatal("oversize")
	}
}

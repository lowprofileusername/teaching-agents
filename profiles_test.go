package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProfilesPersistSwitchAndProtectKeys(t *testing.T) {
	keys := map[string]string{}
	get, set, del := secretGet, secretSet, secretDelete
	defer func() { secretGet, secretSet, secretDelete = get, set, del }()
	secretGet = func(s, k string) (string, error) {
		v, ok := keys[k]
		if !ok {
			return "", errors.New("missing")
		}
		return v, nil
	}
	secretSet = func(s, k, v string) error { keys[k] = v; return nil }
	secretDelete = func(s, k string) error { delete(keys, k); return nil }
	path := filepath.Join(t.TempDir(), "profiles.json")
	a := &App{profiles: &ProfileStore{path: path}}
	call := func(body string, code int) map[string]any {
		t.Helper()
		w := httptest.NewRecorder()
		a.profileAPI(w, httptest.NewRequest("POST", "/", strings.NewReader(body)))
		if w.Code != code {
			t.Fatalf("%d: %s", w.Code, w.Body.String())
		}
		var x map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &x)
		if strings.Contains(w.Body.String(), "secret-A") {
			t.Fatal("secret leaked")
		}
		return x
	}
	x := call(`{"action":"save","name":"LAN","base":"http://10.0.0.3:8888/v1","model":"qwen","key":"secret-A"}`, 200)
	id := x["active"].(string)
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "secret-A") {
		t.Fatal("plaintext key on disk")
	}
	y := call(`{"action":"save","name":"Local","base":"http://localhost:1234/v1","model":"other","clearKey":true}`, 200)
	if a.cfg.Key != "" || a.cfg.Model != "other" {
		t.Fatal("keyless selection failed")
	}
	call(`{"action":"activate","id":"`+id+`"}`, 200)
	if a.cfg.Key != "secret-A" {
		t.Fatal("wrong selected key")
	}
	call(`{"action":"save","id":"`+id+`","name":"LAN renamed","base":"http://10.0.0.3:8888/v1","model":"new-model","thinkingMode":"enabled","thinkingFormat":"enable_thinking","maxTokens":8192,"contextWindow":32768}`, 200)
	if a.cfg.Key != "secret-A" {
		t.Fatal("blank edit erased key")
	}
	call(`{"action":"save","id":"`+id+`","name":"wrong","base":"http://elsewhere/v1","model":"new-model","thinkingMode":"enabled","thinkingFormat":"enable_thinking","maxTokens":8192,"contextWindow":32768}`, 400)
	t.Setenv("TEACHING_AGENT_PROFILES", path)
	b := &App{}
	b.initProfiles()
	if b.cfg.Model != "new-model" || b.cfg.Key != "secret-A" || b.cfg.ThinkingMode != "enabled" || b.cfg.MaxTokens != 8192 {
		t.Fatal("restart did not restore")
	}
	call(`{"action":"delete","id":"`+y["active"].(string)+`"}`, 200)
	call(`{"action":"delete","id":"`+id+`"}`, 200)
	if a.cfg.Base != "" || len(keys) != 0 {
		t.Fatal("delete did not clear active credential")
	}
}
func TestKeyringFailureDoesNotOverwrite(t *testing.T) {
	old := secretSet
	defer func() { secretSet = old }()
	secretSet = func(string, string, string) error { return errors.New("locked") }
	path := filepath.Join(t.TempDir(), "p.json")
	a := &App{profiles: &ProfileStore{path: path}}
	w := httptest.NewRecorder()
	a.profileAPI(w, httptest.NewRequest("POST", "/", strings.NewReader(`{"action":"save","name":"x","base":"http://localhost/v1","model":"x","key":"secret"}`)))
	if w.Code != 500 {
		t.Fatal(w.Code)
	}
	if _, e := os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("wrote failed configuration")
	}
}
func TestLocalModelWithoutKey(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("unexpected auth")
		}
		if r.URL.Path == "/v1/models" {
			_, _ = w.Write([]byte(`{"data":[{"id":"local-qwen"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"local response"}}]}`))
	}))
	defer mock.Close()
	a := &App{cfg: Config{Base: mock.URL + "/v1", Model: "local-qwen"}, client: mock.Client()}
	w := httptest.NewRecorder()
	a.chat(w, httptest.NewRequest("POST", "/", strings.NewReader(`{"question":"test"}`)))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	w = httptest.NewRecorder()
	a.testAPI(w, httptest.NewRequest("POST", "/", strings.NewReader(`{}`)))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"modelFound":true`) {
		t.Fatal(w.Body.String())
	}
}
func TestSystemCredentialStore(t *testing.T) {
	if os.Getenv("TEACHING_TEST_KEYRING") != "1" {
		t.Skip("system keyring probe disabled")
	}
	id := "test-" + newID()
	value := "synthetic-test-" + newID()
	if e := secretSet(keyService, id, value); e != nil {
		t.Fatal("credential store write failed", e)
	}
	defer secretDelete(keyService, id)
	got, e := secretGet(keyService, id)
	if e != nil || got != value {
		t.Fatal("credential store roundtrip failed")
	}
}

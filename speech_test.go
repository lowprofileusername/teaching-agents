package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVoicePackRejectsPathsAndIncompleteInstallation(t *testing.T) {
	for _, p := range []string{"", "../outside", "/absolute"} {
		if _, e := packFile(t.TempDir(), p); e == nil {
			t.Fatalf("accepted %q", p)
		}
	}
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "voice-pack.json"), []byte(`{"version":1,"backend":"mlx","python":"../python","model":"models"}`), 0600)
	if _, e := readVoicePack(root); e == nil {
		t.Fatal("incomplete pack accepted")
	}
}
func TestLocalVoiceAuthenticationValidationAndConcurrency(t *testing.T) {
	a := &App{origin: "http://127.0.0.1:9999", token: "test"}
	send := func(body, token string) int {
		r := httptest.NewRequest("POST", a.origin+"/api/local-voice", strings.NewReader(body))
		r.Header.Set("X-Session-Token", token)
		w := httptest.NewRecorder()
		a.handler().ServeHTTP(w, r)
		return w.Code
	}
	if got := send(`{"Action":"status"}`, "bad"); got != 403 {
		t.Fatal(got)
	}
	if got := send(`{"Action":"synthesize","Text":"你好","Template":"lijing"}`, "test"); got != 400 {
		t.Fatal(got)
	}
	if got := send(`{"Action":"synthesize","Text":"`+strings.Repeat("字", 241)+`","Template":"female"}`, "test"); got != 400 {
		t.Fatal(got)
	}
	a.speech.gate.Lock()
	if got := send(`{"Action":"synthesize","Text":"你好","Template":"female"}`, "test"); got != 409 {
		t.Fatal(got)
	}
	if got := send(`{"Action":"status"}`, "test"); got != 200 {
		t.Fatal(got)
	}
	a.speech.gate.Unlock()
}
func TestSpeechEngineSettingsAndRequestIsolation(t *testing.T) {
	oldGet, oldSet := secretGet, secretSet
	defer func() { secretGet = oldGet; secretSet = oldSet }()
	keys := map[string]string{}
	secretSet = func(service, user, key string) error { keys[user] = key; return nil }
	secretGet = func(service, user string) (string, error) { return keys[user], nil }
	a := &App{profiles: &ProfileStore{path: filepath.Join(t.TempDir(), "profiles.json")}, client: &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}}
	// Fake provider tests wire format only; no paid or real API calls.
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		b, _ := io.ReadAll(r.Body)
		if calls == 1 {
			if r.URL.Path != "/v1/audio/speech" || r.Header.Get("Authorization") != "Bearer private-test" || !strings.Contains(string(b), `"voice":"other-man"`) {
				t.Errorf("bad compatible request %s %s", r.URL.Path, b)
			}
		} else {
			if r.URL.Path != "/cognitiveservices/v1" || r.Header.Get("Ocp-Apim-Subscription-Key") != "azure-test" || !strings.Contains(string(b), "&lt;example&gt;") {
				t.Errorf("bad azure request %s %s", r.URL.Path, b)
			}
		}
		wav := make([]byte, 44)
		copy(wav, "RIFF")
		w.Header().Set("Content-Type", "audio/wav")
		w.Write(wav)
	}))
	defer server.Close()
	if e := a.saveVoiceSettings(voiceRequest{Engine: "compatible", Provider: voiceProvider{Base: server.URL + "/v1", Model: "test-tts", Female: "other-woman", Male: "other-man"}, Key: "private-test"}); e != nil {
		t.Fatal(e)
	}
	s := a.readVoiceSettings()
	w := httptest.NewRecorder()
	a.remoteSpeech(w, httptest.NewRequest("POST", "/", nil), s.Engine, s.Providers[s.Engine], "hello", voiceFor(s.Providers[s.Engine], "male"))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if e := a.saveVoiceSettings(voiceRequest{Engine: "azure", Provider: voiceProvider{Base: server.URL, Female: "zh-CN-XiaoxiaoNeural", Male: "zh-CN-YunxiNeural"}, Key: "azure-test"}); e != nil {
		t.Fatal(e)
	}
	s = a.readVoiceSettings()
	w = httptest.NewRecorder()
	a.remoteSpeech(w, httptest.NewRequest("POST", "/", nil), s.Engine, s.Providers[s.Engine], "<example>", s.Providers[s.Engine].Female)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if e := a.saveVoiceSettings(voiceRequest{Engine: "qwen", Provider: voiceProvider{Female: "Vivian", Male: "Dylan"}}); e != nil {
		t.Fatal(e)
	}
	s = a.readVoiceSettings()
	if s.Providers["compatible"].Model != "test-tts" || s.Providers["azure"].Base != server.URL || s.Providers["qwen"].Female != "Vivian" {
		t.Fatal("engine settings lost")
	}
	w = httptest.NewRecorder()
	a.voiceStatus(w)
	if strings.Contains(w.Body.String(), "private-test") || strings.Contains(w.Body.String(), "azure-test") || strings.Contains(w.Body.String(), "key_ref") {
		t.Fatal("credentials leaked")
	}
	b, _ := os.ReadFile(a.voiceSettingsPath())
	if strings.Contains(string(b), "private-test") || strings.Contains(string(b), "azure-test") {
		t.Fatal("plaintext credentials persisted")
	}
}

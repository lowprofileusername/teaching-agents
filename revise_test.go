package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestReviseInstructionRules(t *testing.T) {
	for _, s := range []string{"修订+批注", "字数只做提醒", "待补：需用户提供的真实证据", "不得为凑4000字自动删减", "revisions", "start==end", "不要求独立成章"} {
		if !strings.Contains(reviseInstruction, s) {
			t.Errorf("revise instruction missing %q", s)
		}
	}
}

func TestReviseModeGuardsAndPrompt(t *testing.T) {
	a := &App{client: &http.Client{Timeout: time.Second * 3}, cfg: Config{Base: "http://127.0.0.1:0", Model: "mock"}}
	for _, c := range []Chat{
		{Mode: "generate", SpecialistID: "architect", Revise: true},
		{Revise: true, StructuredReview: true},
		{Revise: true, StructureReview: true},
	} {
		b, _ := json.Marshal(c)
		w := httptest.NewRecorder()
		a.chat(w, httptest.NewRequest("POST", "/api/chat", strings.NewReader(string(b))))
		if w.Code != 400 {
			t.Fatalf("conflicting revise request accepted: %+v -> %d", c, w.Code)
		}
	}
	calls := 0
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var payload struct {
			Messages []Message `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&payload)
		joined := ""
		for _, m := range payload.Messages {
			joined += m.Content
		}
		if !strings.Contains(joined, "修订+批注") || !strings.Contains(joined, "字数只做提醒") {
			t.Error("revise system prompt not sent to provider")
		}
		jsonOut(w, 200, map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": `{"items":[]}`}, "finish_reason": "stop"}}})
	}))
	defer mock.Close()
	a.cfg = Config{Base: mock.URL, Model: "mock", ModelOptions: ModelOptions{ContextWindow: 32768, MaxTokens: 4096}}
	c := Chat{Revise: true, SpecialistID: "report", Question: "修订这一段", Material: `{"parts":[{"id":"p1@0:4","text":"原文片段"}]}`}
	b, _ := json.Marshal(c)
	w := httptest.NewRecorder()
	a.chat(w, httptest.NewRequest("POST", "/api/chat", strings.NewReader(string(b))))
	if w.Code != 200 || calls != 1 {
		t.Fatal(w.Code, calls, w.Body.String())
	}
}
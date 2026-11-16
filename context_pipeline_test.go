package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestChatActuallySendsEvidenceAndMaterialPresence(t *testing.T) {
	calls := 0
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var payload struct {
			Messages []Message `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&payload)
		got := payload.Messages[1].Content
		if !strings.Contains(got, "宋晓菲") || !strings.Contains(got, "李静") || strings.Contains(got, "预算不足以提供完整检索证据") {
			t.Error("provider did not receive source evidence")
		}
		if !strings.Contains(payload.Messages[0].Content, "本轮没有提供用户上传材料") {
			t.Error("missing material-presence instruction")
		}
		jsonOut(w, 200, map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": "测试回答"}}}})
	}))
	defer mock.Close()
	a := &App{client: &http.Client{Timeout: time.Second}, cfg: Config{Base: mock.URL, Model: "mock", ModelOptions: ModelOptions{ContextWindow: 32768, MaxTokens: 4096}}}
	body, _ := json.Marshal(Chat{Question: "人工智能赛道与普通教学创新有什么区别", Evidence: richEvidenceFixture()})
	w := httptest.NewRecorder()
	a.chat(w, httptest.NewRequest("POST", "/api/chat", strings.NewReader(string(body))))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var result struct {
		Context struct {
			Included int `json:"included_sources"`
		} `json:"context"`
	}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.Context.Included == 0 || calls != 1 {
		t.Fatal("empty evidence reported")
	}
	a.cfg.ContextWindow = 4096
	a.cfg.MaxTokens = 2048
	w = httptest.NewRecorder()
	a.chat(w, httptest.NewRequest("POST", "/api/chat", strings.NewReader(string(body))))
	if w.Code != 400 || calls != 1 {
		t.Fatal("insufficient budget called model", w.Code, calls)
	}
}

func TestStructuredReviewAvoidsConflictingMarkdownInstructions(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Messages []Message `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&payload)
		sys := payload.Messages[0].Content
		if !strings.Contains(sys, "只输出一个完整JSON") || strings.Contains(sys, "在末尾“资料依据”列") {
			t.Error("conflicting output instructions", sys)
		}
		if !strings.Contains(sys, "李静") || !strings.Contains(sys, "advisory_focus") {
			t.Error("expert lens lost in review")
		}
		jsonOut(w, 200, map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": `{"items":[]}`}}}})
	}))
	defer mock.Close()
	a := &App{client: &http.Client{Timeout: time.Second}, cfg: Config{Base: mock.URL, Model: "mock", ModelOptions: ModelOptions{ContextWindow: 32768, MaxTokens: 4096}}}
	body, _ := json.Marshal(Chat{Question: "请审阅", Material: `{"parts":[{"id":"p1","text":"学习证据"}]}`, StructuredReview: true, RequireFullMaterial: true, ExpertID: "lijing"})
	w := httptest.NewRecorder()
	a.chat(w, httptest.NewRequest("POST", "/api/chat", strings.NewReader(string(body))))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestSinglePageDoesNotInheritWholeDeckInstructions(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p struct {
			Messages []Message `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&p)
		if !strings.Contains(p.Messages[0].Content, "本轮范围锁定") {
			t.Error("missing page scope")
		}
		for _, m := range p.Messages {
			if strings.Contains(m.Content, "OLD_WHOLE_DECK") {
				t.Error("inherited previous full deck task")
			}
		}
		jsonOut(w, 200, map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": "本页建议"}}}})
	}))
	defer mock.Close()
	a := &App{client: &http.Client{Timeout: time.Second}, cfg: Config{Base: mock.URL, Model: "mock", ModelOptions: ModelOptions{ContextWindow: 32768, MaxTokens: 4096}}}
	b, _ := json.Marshal(Chat{Question: "这一页怎么样", Material: "幻灯片第19页", AnswerScope: "single_unit", History: []Message{{"user", "OLD_WHOLE_DECK"}}})
	w := httptest.NewRecorder()
	a.chat(w, httptest.NewRequest("POST", "/api/chat", strings.NewReader(string(b))))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
}

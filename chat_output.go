package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type modelMessage struct {
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content"`
	Reasoning        string `json:"reasoning"`
}
type modelResponse struct {
	Choices []struct {
		Delta        modelMessage `json:"delta"`
		Message      modelMessage `json:"message"`
		FinishReason string       `json:"finish_reason"`
	} `json:"choices"`
	Error json.RawMessage `json:"error"`
}

// Split only text actually supplied by the provider. A partial opening or
// closing tag is withheld until its next chunk arrives, never shown as prose.
func splitThinking(s string, final bool) (string, string) {
	var answer, reason strings.Builder
	thinking := false
	for len(s) > 0 {
		marker := "<think>"
		if thinking {
			marker = "</think>"
		}
		i := strings.Index(s, marker)
		if i < 0 {
			keep := len(s)
			if !final {
				for n := 1; n < len(marker) && n <= len(s); n++ {
					if strings.HasSuffix(s, marker[:n]) {
						keep = len(s) - n
					}
				}
			}
			if thinking {
				reason.WriteString(s[:keep])
			} else {
				answer.WriteString(s[:keep])
			}
			break
		}
		if thinking {
			reason.WriteString(s[:i])
		} else {
			answer.WriteString(s[:i])
		}
		s = s[i+len(marker):]
		thinking = !thinking
	}
	return strings.TrimSpace(answer.String()), strings.TrimSpace(reason.String())
}
func reasoningOf(m modelMessage) string {
	if m.ReasoningContent != "" {
		return m.ReasoningContent
	}
	return m.Reasoning
}
func responseText(content, reason string, final bool) (string, string) {
	a, r := splitThinking(content, final)
	if reason == "" {
		reason = r
	}
	return a, reason
}

func chatResult(answer, reason, finish string, meta map[string]any, elapsed time.Duration) map[string]any {
	return map[string]any{"answer": answer, "reasoning": reason, "reasoning_available": reason != "", "reasoning_note": "仅展示接口主动返回的思考文本；不是核验依据，也不代表可获得模型全部内部推理。", "answer_truncated": finish == "length", "context": meta, "model_elapsed_ms": elapsed.Milliseconds()}
}

func (a *App) completeChat(w http.ResponseWriter, r *http.Request, cfg Config, payload map[string]any, meta map[string]any, stream bool) {
	start := time.Now()
	body, _ := json.Marshal(payload)
	if !stream {
		b, err := a.post(r.Context(), cfg.Base+"/chat/completions", cfg.Key, "Authorization", "application/json", bytes.NewReader(body))
		if err != nil {
			fail(w, 502, err.Error())
			return
		}
		var out modelResponse
		if json.Unmarshal(b, &out) != nil || len(out.Choices) == 0 {
			fail(w, 502, "API未返回兼容的回答结构")
			return
		}
		ch := out.Choices[0]
		answer, reason := responseText(ch.Message.Content, reasoningOf(ch.Message), true)
		if answer == "" {
			fail(w, 502, "API未返回最终回答；可能仅生成了思考文本，请调整输出预算或思考等级后重试")
			return
		}
		jsonOut(w, 200, chatResult(answer, reason, ch.FinishReason, meta, time.Since(start)))
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		fail(w, 500, "本机接口不支持流式传输")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("X-Accel-Buffering", "no")
	emit := func(event string, data any) bool {
		b, _ := json.Marshal(data)
		_, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
		flusher.Flush()
		return err == nil
	}
	emit("status", map[string]any{"message": "正在连接模型接口…", "context": meta})
	failStream := func(msg string) { emit("error", map[string]string{"error": msg}) }
	req, err := http.NewRequestWithContext(r.Context(), "POST", cfg.Base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		failStream("模型接口地址无效")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if cfg.Key != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Key)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		failStream("无法连接模型接口或等待超时，请检查服务状态")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		failStream(fmt.Sprintf("模型接口返回HTTP %d；若服务不支持流式，请取消“流式输出”后重试", resp.StatusCode))
		return
	}
	emit("status", map[string]string{"message": "已连接模型接口，等待首段输出…"})
	if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		// Some compatible servers ignore stream=true and return ordinary JSON.
		b, err := io.ReadAll(io.LimitReader(resp.Body, 24<<20))
		var out modelResponse
		if err != nil || json.Unmarshal(b, &out) != nil || len(out.Choices) == 0 {
			failStream("接口没有返回有效流式数据或JSON回答")
			return
		}
		ch := out.Choices[0]
		answer, reason := responseText(ch.Message.Content, reasoningOf(ch.Message), true)
		if reason != "" || answer != "" {
			emit("progress", map[string]string{"answer": answer, "reasoning": reason})
		}
		if answer == "" {
			failStream("接口仅返回思考或空内容，没有最终回答")
			return
		}
		emit("done", chatResult(answer, reason, ch.FinishReason, meta, time.Since(start)))
		return
	}
	var content, reason strings.Builder
	finish := ""
	done := false
	publish := func(final bool) bool {
		a, r := responseText(content.String(), reason.String(), final)
		return emit("progress", map[string]string{"answer": a, "reasoning": r})
	}
	err = readModelSSE(io.LimitReader(resp.Body, 24<<20), func(data string) error {
		if data == "[DONE]" {
			done = true
			return io.EOF
		}
		var out modelResponse
		if json.Unmarshal([]byte(data), &out) != nil {
			return errors.New("模型流式数据格式无效")
		}
		if len(out.Error) > 0 && string(out.Error) != "null" {
			return errors.New("模型服务在生成途中返回错误")
		}
		if len(out.Choices) == 0 {
			return nil
		}
		ch := out.Choices[0]
		content.WriteString(ch.Delta.Content)
		reason.WriteString(reasoningOf(ch.Delta))
		if ch.FinishReason != "" {
			finish = ch.FinishReason
		}
		if content.Len()+reason.Len() > 2<<20 {
			return errors.New("模型输出超过本次显示上限")
		}
		if !publish(false) {
			return errors.New("前端连接已关闭")
		}
		return nil
	})
	if err != nil {
		if r.Context().Err() == nil {
			failStream(err.Error())
		}
		return
	}
	answer, thinking := responseText(content.String(), reason.String(), true)
	if !done && finish == "" {
		failStream("模型连接中断，已显示的内容不是完整回答")
		return
	}
	if answer == "" {
		failStream("模型只返回了思考文本，没有最终回答；请增加输出预算或降低思考等级")
		return
	}
	emit("done", chatResult(answer, thinking, finish, meta, time.Since(start)))
}

func readModelSSE(r io.Reader, onData func(string) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 2<<20)
	data := []string{}
	dispatch := func() error {
		if len(data) == 0 {
			return nil
		}
		s := strings.Join(data, "\n")
		data = nil
		return onData(s)
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := dispatch(); err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
		} else if strings.HasPrefix(line, "data:") {
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if err := scanner.Err(); err != nil {
		return errors.New("读取模型流式输出失败")
	}
	if err := dispatch(); err != nil && err != io.EOF {
		return err
	}
	return nil
}

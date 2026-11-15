package main

import (
	"errors"
	"strings"
)

type ModelOptions struct {
	ThinkingMode    string `json:"thinkingMode"`
	ThinkingFormat  string `json:"thinkingFormat"`
	ReasoningEffort string `json:"reasoningEffort"`
	ContextWindow   int    `json:"contextWindow"`
	MaxTokens       int    `json:"maxTokens"`
}

func (o *ModelOptions) normalize() error {
	if o.ThinkingMode == "" {
		o.ThinkingMode = "default"
	}
	if o.ThinkingFormat == "" {
		o.ThinkingFormat = "default"
	}
	if o.ContextWindow == 0 {
		o.ContextWindow = 32768
	}
	if o.MaxTokens == 0 {
		o.MaxTokens = 4096
	}
	if !oneOf(o.ThinkingMode, "default", "enabled", "disabled") || !oneOf(o.ThinkingFormat, "default", "reasoning_effort", "enable_thinking", "thinking", "chat_template_kwargs") || !oneOf(o.ReasoningEffort, "", "none", "minimal", "low", "medium", "high", "max", "xhigh") {
		return errors.New("思考参数无效")
	}
	if o.ThinkingMode != "default" && o.ThinkingFormat == "default" {
		return errors.New("请选择接口支持的思考参数格式")
	}
	if o.ContextWindow < 4096 || o.ContextWindow > 2000000 || o.MaxTokens < 256 || o.MaxTokens > 131072 || o.MaxTokens > o.ContextWindow/2 {
		return errors.New("上下文需4096–2000000，输出上限需256–131072且不超过上下文的一半")
	}
	return nil
}
func oneOf(s string, vs ...string) bool {
	for _, v := range vs {
		if s == v {
			return true
		}
	}
	return false
}
func (o ModelOptions) apply(body map[string]any) {
	_ = o.normalize()
	body["max_tokens"] = o.MaxTokens
	if o.ReasoningEffort != "" {
		body["reasoning_effort"] = o.ReasoningEffort
	}
	if o.ThinkingMode == "default" {
		return
	}
	on := o.ThinkingMode == "enabled"
	switch o.ThinkingFormat {
	case "enable_thinking":
		body["enable_thinking"] = on
	case "thinking":
		body["thinking"] = map[string]string{"type": o.ThinkingMode}
	case "chat_template_kwargs":
		body["chat_template_kwargs"] = map[string]bool{"enable_thinking": on}
	case "reasoning_effort":
		if !on {
			body["reasoning_effort"] = "none"
		} else if o.ReasoningEffort == "" || o.ReasoningEffort == "none" {
			body["reasoning_effort"] = "high"
		}
	}
}
func bounded(s string, n int) string {
	rs := []rune(s)
	if n < 0 {
		n = 0
	}
	if len(rs) <= n {
		return s
	}
	return string(rs[:n]) + "\n〔上下文预算已截断，未提供剩余部分〕"
}
func stripThink(s string) string {
	for {
		a := strings.Index(s, "<think>")
		if a < 0 {
			return s
		}
		b := strings.Index(s[a:], "</think>")
		if b < 0 {
			return s[:a]
		}
		s = s[:a] + s[a+b+len("</think>"):]
	}
}

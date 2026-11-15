package main

import (
	"encoding/json"
)

// Keep valid JSON and all named sources when the context budget is small.
func fitEvidence(raw string, budget int) string {
	if len([]rune(raw)) <= budget {
		return raw
	}
	var root map[string]any
	if json.Unmarshal([]byte(raw), &root) != nil {
		return bounded(raw, budget)
	}
	for _, key := range []string{"raw_ocr_unverified", "official_pdf_hits", "research_judgments"} {
		if a, ok := root[key].([]any); ok && len(a) > 2 {
			root[key] = a[:2]
		}
	}
	if pages, ok := root["pages"].([]any); ok {
		for _, p := range pages {
			if m, ok := p.(map[string]any); ok {
				delete(m, "research_analysis")
				delete(m, "issues")
				delete(m, "annotations")
				if a, ok := m["transcript_sample_unverified"].([]any); ok && len(a) > 2 {
					m["transcript_sample_unverified"] = a[:2]
				}
				if a, ok := m["official_pages"].([]any); ok && len(a) > 1 {
					m["official_pages"] = a[:1]
				}
			}
		}
	}
	root["context_note"] = "仅提供各来源摘要片段；保留来源编号，不代表已提供整页/整场。"
	preserve := map[string]bool{"id": true, "document_id": true, "page_id": true, "tag": true, "teacher": true, "title": true, "citation": true, "authority_status": true, "institution": true, "course": true, "original_page_no": true}
	var trim func(any, string, int) any
	trim = func(v any, key string, n int) any {
		switch x := v.(type) {
		case string:
			if preserve[key] {
				return x
			}
			if len([]rune(x)) > n {
				return string([]rune(x)[:n]) + "〔截取〕"
			}
			return x
		case []any:
			a := make([]any, len(x))
			for i, t := range x {
				a[i] = trim(t, key, n)
			}
			return a
		case map[string]any:
			m := map[string]any{}
			for k, t := range x {
				m[k] = trim(t, k, n)
			}
			return m
		default:
			return v
		}
	}
	for _, n := range []int{1200, 800, 500, 300, 200, 100, 50} {
		b, _ := json.Marshal(trim(root, "", n))
		if len([]rune(string(b))) <= budget {
			return string(b)
		}
	}
	return `{"context_note":"上下文预算不足以提供完整检索证据；请缩小问题范围，不能据此编造回答。"}`
}

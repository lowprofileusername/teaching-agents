package main

import (
	"encoding/json"
	"fmt"
)

type evidenceSource struct {
	fields map[string]any
	group  string
}

func sourceCount(raw string) int {
	var root map[string]any
	if json.Unmarshal([]byte(raw), &root) != nil {
		return 0
	}
	n := 0
	for _, key := range []string{"sources", "pages", "document_evidence", "raw_ocr_unverified", "official_pdf_hits", "research_judgments"} {
		if a, ok := root[key].([]any); ok {
			n += len(a)
		}
	}
	return n
}
func evidenceSources(root map[string]any) []evidenceSource {
	out := []evidenceSource{}
	add := func(kind string, r map[string]any, fields []string, group string) {
		m := map[string]any{"kind": kind}
		for _, k := range fields {
			if v, ok := r[k]; ok && v != nil && v != "" {
				m[k] = v
			}
		}
		out = append(out, evidenceSource{m, group})
	}
	for _, key := range []string{"pages", "document_evidence", "official_pdf_hits", "raw_ocr_unverified", "research_judgments"} {
		a, _ := root[key].([]any)
		for _, v := range a {
			r, ok := v.(map[string]any)
			if !ok {
				continue
			}
			switch key {
			case "pages":
				add("page", r, []string{"id", "teacher", "institution", "course", "title", "original_page_no", "page_facts"}, "speaker:"+fmt.Sprint(r["teacher"]))
				m := out[len(out)-1].fields
				if a, ok := r["official_pages"].([]any); ok && len(a) > 0 {
					m["official_page_reference"] = a[0]
				}
				if a, ok := r["occurrences"].([]any); ok && len(a) > 0 {
					m["occurrence_example"] = a[0]
				}
				if a, ok := r["transcript_sample_unverified"].([]any); ok && len(a) > 0 {
					if u, ok := a[0].(map[string]any); ok {
						sample := map[string]any{}
						for _, k := range []string{"text", "t0", "t1", "version", "quality_status", "teacher_attribution"} {
							if v, ok := u[k]; ok {
								sample[k] = v
							}
						}
						m["transcript_unverified_excerpt"] = sample
					}
				}
				for _, k := range []string{"issues", "annotations"} {
					if a, ok := r[k].([]any); ok && len(a) > 0 {
						m["review_issues_present"] = true
					}
				}
			case "document_evidence":
				add("research_or_rule_pdf", r, []string{"id", "document_id", "title", "pdf_page_no", "authority_status", "edition", "extraction", "text"}, "document:"+fmt.Sprint(r["document_id"]))
			case "official_pdf_hits":
				add("official_pdf", r, []string{"id", "teacher", "title", "pdf_page_no", "direct_text", "ocr_text"}, "official:"+fmt.Sprint(r["id"]))
			case "raw_ocr_unverified":
				add("raw_ocr_unverified", r, []string{"tag", "text_id", "seconds", "text"}, "ocr:"+fmt.Sprint(r["tag"]))
			case "research_judgments":
				add("research_judgment", r, []string{"id", "title", "payload_json"}, "research:"+fmt.Sprint(r["id"]))
			}
		}
	}
	// One page for every speaker comes before extra pages from the same speaker.
	ordered := []evidenceSource{}
	used := map[int]bool{}
	groups := map[string]bool{}
	for i, s := range out {
		if s.fields["kind"] == "page" && !groups[s.group] {
			ordered = append(ordered, s)
			used[i] = true
			groups[s.group] = true
		}
	}
	for i, s := range out {
		if !used[i] && s.fields["kind"] != "page" && !groups[s.group] {
			ordered = append(ordered, s)
			used[i] = true
			groups[s.group] = true
		}
	}
	for i, s := range out {
		if !used[i] {
			ordered = append(ordered, s)
		}
	}
	return ordered
}
func clipEvidence(v any, key string, n int) any {
	switch x := v.(type) {
	case string:
		limit := n
		switch key {
		case "id", "document_id", "teacher", "tag", "original_page_no", "authority_status", "edition", "extraction", "kind":
			return x
		case "title":
			if limit > 160 {
				limit = 160
			}
		}
		if len([]rune(x)) > limit {
			return string([]rune(x)[:limit]) + "〔摘录〕"
		}
		return x
	case map[string]any:
		m := map[string]any{}
		for k, v := range x {
			m[k] = clipEvidence(v, k, n)
		}
		return m
	case []any:
		a := make([]any, len(x))
		for i, v := range x {
			a[i] = clipEvidence(v, key, n)
		}
		return a
	default:
		return v
	}
}
func packSources(s []any, total int) string {
	b, _ := json.Marshal(map[string]any{"sources": s, "retrieved_source_count": total, "included_source_count": len(s), "context_note": "以下为实际提供的证据摘录，不是整场/全库全文。未提供的部分不能推断为没有资料；机器转写未听校。有核查标记的页须回看来源。"})
	return string(b)
}
func fitEvidence(raw string, budget int) string {
	if len([]rune(raw)) <= budget {
		return raw
	}
	var root map[string]any
	if json.Unmarshal([]byte(raw), &root) != nil {
		return bounded(raw, budget)
	}
	sources := evidenceSources(root)
	for _, n := range []int{1200, 600, 300, 160, 80} {
		parts := []any{}
		for _, s := range sources {
			parts = append(parts, clipEvidence(s.fields, "", n))
		}
		packed := packSources(parts, len(sources))
		if len([]rune(packed)) <= budget {
			return packed
		}
	}
	// Metadata can exceed the budget even after text trimming. Keep useful source
	// records instead of replacing every record with a "budget insufficient" note.

	best := []any{}
	for _, size := range []int{120, 80, 40} {
		chosen := []any{}
		for _, s := range sources {
			m := map[string]any{}
			for _, k := range []string{"kind", "id", "document_id", "teacher", "title", "pdf_page_no", "authority_status", "extraction", "page_facts", "text", "direct_text", "ocr_text", "payload_json", "review_issues_present"} {
				if v, ok := s.fields[k]; ok {
					m[k] = v
				}
			}
			m = clipEvidence(m, "", size).(map[string]any)
			candidate := append(append([]any{}, chosen...), m)
			if len([]rune(packSources(candidate, len(sources)))) <= budget {
				chosen = candidate
			}
		}
		if len(chosen) > len(best) {
			best = chosen
		}
	}
	return packSources(best, len(sources))
}

package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestEvidenceBudgetKeepsComparisonAndJSON(t *testing.T) {
	x := map[string]any{"pages": []any{map[string]any{"id": "songxiaofei:R12", "teacher": "宋晓菲", "page_facts": strings.Repeat("教学机制", 3000)}, map[string]any{"id": "lijing:R12", "teacher": "李静", "page_facts": strings.Repeat("学生学习", 3000)}}}
	b, _ := json.Marshal(x)
	got := fitEvidence(string(b), 1800)
	var out any
	if json.Unmarshal([]byte(got), &out) != nil || len([]rune(got)) > 1800 || !strings.Contains(got, "songxiaofei:R12") || !strings.Contains(got, "lijing:R12") {
		t.Fatal("evidence sources lost", got)
	}
}

func richEvidenceFixture() string {
	pages := []any{}
	for i := 0; i < 8; i++ {
		teacher, id := "宋晓菲", "songxiaofei"
		if i%2 == 1 {
			teacher, id = "李静", "lijing"
		}
		occ := []any{}
		for n := 0; n < 400; n++ {
			occ = append(occ, map[string]any{"t0": n, "t1": n + 1})
		}
		pages = append(pages, map[string]any{"id": fmt.Sprintf("%s:R%02d", id, i+1), "teacher": teacher, "title": strings.Repeat("课程标题", 50), "page_facts": strings.Repeat("学生任务与评价证据", 100), "occurrences": occ, "issues": []any{map[string]any{"issue": "转写尚未听校"}}})
	}
	docs := []any{}
	for n := 0; n < 6; n++ {
		docs = append(docs, map[string]any{"id": fmt.Sprintf("doc%d:P1", n), "document_id": fmt.Sprintf("doc%d", n), "title": strings.Repeat("单视频研究报告", 35), "pdf_page_no": 1, "authority_status": "research_analysis_not_primary_evidence", "extraction": "pdf_text_layer", "text": strings.Repeat("教学创新机制分析", 200)})
	}
	b, _ := json.Marshal(map[string]any{"pages": pages, "document_evidence": docs})
	return string(b)
}
func TestRichEvidenceNeverDropsAllAndStaysWithinBudget(t *testing.T) {
	for _, budget := range []int{850, 1800, 4000, 7000} {
		s := fitEvidence(richEvidenceFixture(), budget)
		if !json.Valid([]byte(s)) || len([]rune(s)) > budget || sourceCount(s) == 0 {
			t.Fatalf("budget %d: %s", budget, s)
		}
		if !strings.Contains(s, "宋晓菲") || !strings.Contains(s, "李静") {
			t.Fatalf("lost comparison at budget %d: %s", budget, s)
		}
	}
}

package main

import (
	"encoding/json"
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

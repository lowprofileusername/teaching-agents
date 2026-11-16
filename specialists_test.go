package main

import (
	"strings"
	"testing"
)

func TestTeachingSpecialistsUseDistinctChecksAndTrack(t *testing.T) {
	report, e := specialistPrompt("auto", "report", "新文科")
	if e != nil || !strings.Contains(report, "段落位置") || !strings.Contains(report, "人文社会") {
		t.Fatal(report, e)
	}
	lesson, _ := specialistPrompt("auto", "lesson", "人工智能")
	if !strings.Contains(lesson, "教师提问") || !strings.Contains(lesson, "人机分工") || strings.Contains(lesson, "段落位置") {
		t.Fatal(lesson)
	}
	explicit, _ := specialistPrompt("evidence", "report", "新医科")
	if !strings.Contains(explicit, "分母") || !strings.Contains(explicit, "临床推理") {
		t.Fatal(explicit)
	}
	if _, e = specialistPrompt("made-up", "report", ""); e == nil {
		t.Fatal("unknown specialist accepted")
	}
}

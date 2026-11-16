package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestAITrackBuiltinFourSourcesAndEdition(t *testing.T) {
	if _, e := builtinLibrary(); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct{ q, file, want string }{
		{"第七届 AI 赛道报告评分维度有哪些", "rules_7_ai_2026.json", "非国赛正式标准"},
		{"第六届 AI 赛道一等奖有哪些课程和做法", "ai_award_cases_6th.json", "不是你的课程事实"},
		{"AI 赛道怎么防止学生让 AI 代做", "ai_playbook.json", "非专家本人观点"},
		{"第五届新文科课堂视频资源下载", "video_index_5th_wenke.json", "未观看"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			found := builtinSearch(context.Background(), tc.q)
			count := 0
			for _, r := range found {
				if strings.Contains(stringField(r, "source_file"), tc.file) {
					count++
					if !strings.Contains(stringField(r, "authority_status"), tc.want) || stringField(r, "source_url") == "" {
						t.Errorf("missing source boundary: %v", r)
					}
				}
				if strings.Contains(tc.q, "AI") && r["kind"] == "award_case" && strings.Contains(stringField(r, "title"), "非AI赛道") {
					t.Fatal("non-AI-track case returned for generic AI query")
				}
				if r["kind"] == "development_reference" {
					t.Fatal("developer task retrieved as competition advice")
				}
			}
			if count == 0 {
				t.Fatalf("file not retrieved: %s", tc.file)
			}
			if strings.Contains(tc.q, "第六届") {
				b, _ := json.Marshal(found)
				for _, s := range []string{"宋晓菲", "酶工程", "李静", "流体力学", "辛惠娟", "水力学"} {
					if !strings.Contains(string(b), s) {
						t.Errorf("missing award result %s", s)
					}
				}
			}
		})
	}
	for _, r := range builtinRuleSearch(context.Background(), "第六届人工智能报告评分") {
		if r["edition"] == 7 {
			t.Fatal("7th rules mislabeled sixth")
		}
	}
	if len(builtinRuleSearch(context.Background(), "第六届人工智能报告评分")) == 0 {
		t.Fatal("old rule retrieval broken")
	}
	if len(builtinRuleSearch(context.Background(), "第七届AI报告评分")) != 0 {
		t.Fatal("old rules used for 7th")
	}
	db, e := builtinLibrary()
	if e != nil {
		t.Fatal(e)
	}
	for kind, want := range map[string]int{"provisional_rule": 4, "ai_migration": 12, "award_case": 10, "video_index": 24} {
		var n int
		if e = db.QueryRow("SELECT COUNT(*) FROM builtin_reference WHERE kind=?", kind).Scan(&n); e != nil || n != want {
			t.Errorf("%s count %d want %d (%v)", kind, n, want, e)
		}
	}
	if _, e = db.Exec("DELETE FROM builtin_reference"); e == nil {
		t.Fatal("builtin DB is writable")
	}
}

func TestAITrackSpecialistsPreserveDefaults(t *testing.T) {
	if queryEdition("第七届报告，参考第六届做法") != 7 {
		t.Fatal("reference edition overrides requested edition")
	}
	if len(teachingSpecialists) != 10 {
		t.Fatal("expected old eight plus two builders")
	}
	for _, id := range []string{"architect", "video-planner"} {
		p, e := specialistPrompt(id, "report", "人工智能")
		if e != nil || !strings.Contains(p, "迁移") && id == "video-planner" {
			t.Fatal(p, e)
		}
	}
	if len(defaultSpecialists) != 7 {
		t.Fatal("workspace mapping changed")
	}
	if defaultSpecialists["report"] != "report" || defaultSpecialists["lesson"] != "lesson" {
		t.Fatal("checker defaults replaced")
	}
}

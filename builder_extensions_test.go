package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func presentationFixture(n, seconds int) BuilderPlan {
	p := builderFixture("architect")
	p.Kind = "presentation"
	p.WordBudget = 0
	p.Sections = []BuilderSection{}
	p.Slides = []BuilderSlide{}
	for i := 0; i < n; i++ {
		d := seconds / n
		if i == n-1 {
			d += seconds % n
		}
		p.Slides = append(p.Slides, BuilderSlide{ID: strings.Repeat("p", i+1), PageNo: i + 1, Title: "以可核查问题组织教学", Section: "创新内容", Bullets: []string{"待补：学生原始任务记录"}, Visual: "匿名化任务截图", EvidenceSource: "builtin:rules7:scoring", SpeakerNote: "建议从真实问题说明举措与证据的关系", DurationSec: d})
	}
	for i := range p.Scores {
		p.Scores[i].Sections = []string{"p"}
	}
	return p
}
func TestPresentationArchitectPlan(t *testing.T) {
	refs := map[string]map[string]any{"builtin:rules7:scoring": {"authority_status": rules7Notice}}
	for _, tc := range []struct{ n, sec int }{{10, 700}, {31, 700}, {10, 720}, {1, 600}} {
		p, e := validateBuilderPlan(fixtureJSON(presentationFixture(tc.n, tc.sec)), "presentation-architect", refs)
		if e != nil {
			t.Fatal(tc, e)
		}
		if len(p.Warnings) == 0 {
			t.Fatal("soft guidance missing warnings", tc)
		}
	}
	for _, mutate := range []func(*BuilderPlan){
		func(p *BuilderPlan) { p.Slides[0].DurationSec = 721 },
		func(p *BuilderPlan) { p.Slides[0].EvidenceSource = "invented" },
		func(p *BuilderPlan) { p.Slides[0].Title = "郑州大学李静" },
		func(p *BuilderPlan) { p.Slides[0].Bullets = []string{"学生满意度提升32%"} },
		func(p *BuilderPlan) { p.Slides[0].SpeakerNote = "" },
		func(p *BuilderPlan) { p.Slides = nil },
	} {
		p := presentationFixture(1, 700)
		mutate(&p)
		if _, e := validateBuilderPlan(fixtureJSON(p), "presentation-architect", refs); e == nil {
			t.Fatal("bad slide accepted", p)
		}
	}
	if !isBuilder("presentation-architect") {
		t.Fatal("role missing")
	}
	if !strings.Contains(generationPrompt("presentation-architect", 7, "人工智能"), "720秒") {
		t.Fatal("presentation prompt missing")
	}
}
func TestFullDraftBudgetsAndSafety(t *testing.T) {
	refs := map[string]map[string]any{"builtin:rules7:scoring": {"authority_status": rules7Notice}}
	policy, _ := builderPolicyFor("人工智能", 7)
	policy.Depth = "full"
	p := builderFixture("architect")
	for i := range p.Sections {
		p.Sections[i].Draft = "拟从学情记录定位问题，围绕核验设计任务，由学生说明修订依据。待补：真实学情与过程证据。"
	}
	if _, e := validateBuilderPlan(fixtureJSON(p), "architect", refs, policy); e != nil {
		t.Fatal(e)
	}
	for _, draft := range []string{"", "学生成绩提高32%", strings.Repeat("字", 3201), "学生说：这个课很好", "获得全国一等奖"} {
		bad := p
		bad.Sections = append([]BuilderSection{}, p.Sections...)
		bad.Sections[0].Draft = draft
		if _, e := validateBuilderPlan(fixtureJSON(bad), "architect", refs, policy); e == nil {
			t.Fatal("bad draft accepted", len(draft))
		}
	}
	if !strings.Contains(generationPrompt("architect", 7, "新文科", "full"), "本轮depth=full") {
		t.Fatal("depth not in prompt")
	}
	if strings.Contains(generationPrompt("architect", 7, "新文科", "full"), "独立设置AI伦理与治理") {
		t.Fatal("AI chapter forced")
	}
	// Draft validation must not change the ordinary document-review word-count reminder.
	if !strings.Contains(reviseInstruction, "不得因字数拒绝修订") {
		t.Fatal("review reminder changed")
	}
}
func TestTrackPlaybooksTraceability(t *testing.T) {
	raw, e := os.ReadFile("research/source-evidence/02_专家方法库_36场.md")
	if e != nil {
		t.Fatal(e)
	}
	for _, track := range []string{"新工科", "新医科", "新农科", "新文科", "基础课程", "课程思政", "产教融合", "实验教学"} {
		for _, m := range trackPlaybook(track).Plays {
			if !strings.Contains(string(raw), "| "+m.Source+" | "+m.Mechanism+" | "+m.Caution+" |") {
				t.Fatal("caution or mechanism changed from source row", m)
			}
		}
	}

	for track, names := range map[string][]string{"新工科": {"刘学莘", "熊芸"}, "新医科": {"辛页", "姜宏佺"}, "新农科": {"向妙莲"}, "实验教学": {"刘战合"}} {
		p := trackPlaybook(track)
		b, _ := json.Marshal(p)
		for _, name := range names {
			if !strings.Contains(string(b), name) {
				t.Error(track, name)
			}
		}
		found := builtinSearch(context.Background(), track+" 报告打法")
		for _, name := range names {
			matched := false
			for _, r := range found {
				if r["kind"] == "track_migration" && strings.Contains(stringField(r, "origin"), name) {
					matched = true
				}
			}
			if !matched {
				t.Error("track search missing", track, name)
			}
		}
		for _, m := range p.Plays {
			if m.Source == "" || m.Locator == "" || m.Caution == "" || len(m.Dimensions) == 0 {
				t.Fatal("missing provenance", m)
			}
		}
		prompt := generationPrompt("architect", 7, track)
		for _, name := range names {
			if !strings.Contains(prompt, name) {
				t.Error("prompt missing", name)
			}
		}
	}
	if p := trackPlaybook("不存在的赛道"); p.Status != "no_traceable_source" || len(p.Plays) != 0 {
		t.Fatal(p)
	}
	if p := trackPlaybook("人工智能"); p.Status != "reuse_ai_playbook" || len(p.Plays) != 0 {
		t.Fatal("AI duplicated")
	}
}

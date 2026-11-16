package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func builderJPEG() string {
	var b bytes.Buffer
	jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil)
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}
func builderFixture(role string) BuilderPlan {
	p := BuilderPlan{Kind: "report", Line: "以问题核验串联自主判断", Loop: "问题识别到举措再到证据", WordBudget: 4000, Sections: []BuilderSection{{"s1", "课程问题与创新举措", 3200, []string{"哪些问题有真实依据"}, []string{"待补：原始学情记录"}}, {"s2", "AI伦理与治理", 800, []string{"如何核验模型回答"}, []string{"待补：治理规则"}}}, Scores: []BuilderScore{}, Evidence: []string{"待补：改革前后同口径原始记录"}, Timeline: []BuilderStage{}, Demo: []BuilderDemo{}, Annotations: []BuilderAnnotation{}, References: []BuilderReference{{"builtin:rules7:scoring", "借鉴章节对照结构", ""}}, Boundary: "待实施建议"}
	ids := []string{"s1", "s2"}
	if role == "video-planner" {
		p.Kind = "lesson"
		p.WordBudget = 0
		p.Duration = 45
		p.Sections = []BuilderSection{}
		p.Timeline = []BuilderStage{{"t1", 0, 20, "问题识别", "建议让AI追问", "拟拍摄学生说明判断依据", "问题辨析"}, {"t2", 20, 45, "交叉核验", "建议师生核验AI回答", "拟拍摄学生修订过程", "核验修订"}}
		ids = []string{"t1", "t2"}
		p.DemoDuration = 15
		p.Demo = []BuilderDemo{{"d1", 0, 5, "课前", "拟介绍课前问题诊断", "拟展示AI追问", "拟展示问题清单", "t1"}, {"d2", 5, 10, "课中衔接", "拟介绍课堂核验衔接", "拟展示核验记录", "拟展示课堂修订材料", "t2"}, {"d3", 10, 15, "课后", "拟介绍课后延伸", "拟展示AI反馈", "拟展示修订作品", "t2"}}
		p.Annotations = []BuilderAnnotation{{"t1", "引导判断", "用追问诊断", "预期形成判断依据"}, {"t2", "训练核验", "查证后修订", "预期形成修订理由"}}
	}
	for _, dim := range aiReportDimensions() {
		p.Scores = append(p.Scores, BuilderScore{dim, ids, []string{"待补：对应教学过程记录"}})
	}
	return p
}
func fixtureJSON(p BuilderPlan) string { b, _ := json.Marshal(p); return string(b) }
func TestGenerateValidationRejectsUnsupportedFactsAndIncompletePlans(t *testing.T) {
	for _, raw := range []string{"null", "plain text reference"} {
		if !strings.Contains(appendBuiltinEvidence(context.Background(), Chat{Mode: "generate", Edition: "7", Evidence: raw}), "builtin:rules7:scoring") {
			t.Fatal("builtin generation evidence missing")
		}
	}

	if !strings.Contains(generationPrompt("architect", 6), "第六届rules.json") || strings.Contains(generationPrompt("architect", 6), "第七届六维为") {
		t.Fatal("sixth edition generation mislabeled")
	}

	available := map[string]map[string]any{"builtin:rules7:scoring": {"authority_status": rules7Notice}}
	for _, role := range []string{"architect", "video-planner"} {
		p, e := validateBuilderPlan(fixtureJSON(builderFixture(role)), role, available)
		if e != nil || p.Boundary == "" {
			t.Fatal(role, e)
		}
	}
	for _, bad := range []string{"学生满意度提升32%", "覆盖五百人", "学生表示很喜欢", "获得省级奖项", "被同行采用", "已提升学习质量", "学生说：完全理解了"} {
		p := builderFixture("architect")
		p.Line = bad
		if _, e := validateBuilderPlan(fixtureJSON(p), "architect", available); e == nil {
			t.Errorf("fabrication accepted: %s", bad)
		}
	}
	for _, mutate := range []func(*BuilderPlan){
		func(p *BuilderPlan) { p.Sections[0].WordBudget-- }, func(p *BuilderPlan) { p.Sections[1].Title = "普通建议" }, func(p *BuilderPlan) { p.Scores = p.Scores[:5] }, func(p *BuilderPlan) { p.Scores[0].Sections = []string{"missing"} }, func(p *BuilderPlan) { p.References[0].ID = "made-up" }, func(p *BuilderPlan) { p.Evidence = []string{"学习成绩优秀"} },
	} {
		p := builderFixture("architect")
		mutate(&p)
		if _, e := validateBuilderPlan(fixtureJSON(p), "architect", available); e == nil {
			t.Fatal("invalid plan accepted")
		}
	}
	p := builderFixture("video-planner")
	p.Timeline[1].Start = 21
	if _, e := validateBuilderPlan(fixtureJSON(p), "video-planner", available); e == nil {
		t.Fatal("timeline gap accepted")
	}
	missingTime := strings.Replace(fixtureJSON(builderFixture("video-planner")), `"start_minute":0,`, "", 1)
	if _, e := validateBuilderPlan(missingTime, "video-planner", available); e == nil {
		t.Fatal("missing time field accepted as zero")
	}
	raw := fixtureJSON(builderFixture("architect"))
	for _, s := range []string{"```json\n" + raw + "\n```", raw + raw, strings.Replace(raw, `"duration_minutes":0,`, "", 1), strings.Replace(raw, `"duration_minutes":0`, `"duration_minutes":0,"invented_result":"优秀"`, 1)} {
		if _, e := validateBuilderPlan(s, "architect", available); e == nil {
			t.Fatal("invalid JSON accepted")
		}
	}
}
func TestGenerateActualProviderFlowAndCheckerRegression(t *testing.T) {
	for _, role := range []string{"architect", "video-planner"} {
		t.Run(role, func(t *testing.T) {
			calls := 0
			bad := false
			mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				var payload struct {
					Messages []Message `json:"messages"`
					Stream   bool      `json:"stream"`
				}
				json.NewDecoder(r.Body).Decode(&payload)
				joined := ""
				for _, m := range payload.Messages {
					joined += m.Content
				}
				for _, s := range []string{"本轮没有提供用户上传材料", "省赛校赛口径汇总", "ai_playbook.json", "ai_award_cases_6th.json"} {
					if !strings.Contains(joined, s) {
						t.Errorf("provider missing %s", s)
					}
				}
				if strings.Contains(joined, "OLD_TEACHER_FACTS") || payload.Stream {
					t.Error("history leaked or unvalidated stream")
				}
				p := builderFixture(role)
				if bad {
					p.Line = "学生成绩提升32%"
				}
				jsonOut(w, 200, map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": fixtureJSON(p)}, "finish_reason": "stop"}}})
			}))
			defer mock.Close()
			a := &App{client: &http.Client{Timeout: time.Second * 3}, cfg: Config{Base: mock.URL, Model: "mock", ModelOptions: ModelOptions{ContextWindow: 32768, MaxTokens: 4096}}}
			c := Chat{Mode: "generate", SpecialistID: role, Edition: "7", Track: "人工智能", Question: "请设计报告骨架和课堂策划", Stream: true, History: []Message{{"assistant", "OLD_TEACHER_FACTS"}}}
			run := func() *httptest.ResponseRecorder {
				b, _ := json.Marshal(c)
				w := httptest.NewRecorder()
				a.chat(w, httptest.NewRequest("POST", "/api/chat", strings.NewReader(string(b))))
				return w
			}
			w := run()
			if w.Code != 200 || !strings.Contains(w.Body.String(), `"generation_validated":true`) {
				t.Fatal(w.Code, w.Body.String())
			}
			bad = true
			w = run()
			if w.Code != 502 || strings.Contains(w.Body.String(), `"plan"`) {
				t.Fatal("fabrication published", w.Code, w.Body.String())
			}
			c.StructuredReview = true
			w = run()
			if w.Code != 400 || calls != 2 {
				t.Fatal("conflicting modes accepted")
			}
		})
	}
	for _, kind := range []string{"text", "structure", "image"} {
		t.Run(kind, func(t *testing.T) {
			mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var p map[string]any
				json.NewDecoder(r.Body).Decode(&p)
				b, _ := json.Marshal(p["messages"])
				s := string(b)
				if strings.Contains(s, "innovation_line") || !strings.Contains(s, "只输出") || !strings.Contains(s, "非国赛正式标准") {
					t.Error("checker schema or AI references lost")
				}
				jsonOut(w, 200, map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": `{"items":[]}`}}}})
			}))
			defer mock.Close()
			a := &App{client: &http.Client{Timeout: time.Second * 3}, cfg: Config{Base: mock.URL, Model: "mock", ModelOptions: ModelOptions{ContextWindow: 32768, MaxTokens: 4096}}}
			c := Chat{Question: "第七届人工智能报告检查", SpecialistID: "report", Material: "用户材料片段"}
			switch kind {
			case "text":
				c.StructuredReview = true
			case "structure":
				c.StructureReview = true
			case "image":
				c.Images = []VisionImage{{Label: "img1", DataURL: builderJPEG()}}
				c.VisionConfirmed = true
			}
			b, _ := json.Marshal(c)
			w := httptest.NewRecorder()
			a.chat(w, httptest.NewRequest("POST", "/api/chat", strings.NewReader(string(b))))
			if w.Code != 200 {
				t.Fatal(w.Code, w.Body.String())
			}
		})
	}
}

func TestBuilderTrackSwitchAndTwoVideoRequirements(t *testing.T) {
	policy, e := builderPolicyFor("新文科", 7)
	if e != nil || len(policy.Dimensions) != 5 || policy.AI {
		t.Fatal(policy, e)
	}
	c := Chat{Mode: "generate", Track: "新文科", Edition: "7", Question: "请设计报告骨架"}
	evidence := appendBuiltinEvidence(context.Background(), c)
	refs := includedBuilderReferences(evidence)
	p := builderFixture("architect")
	p.Sections[1].Title = "课程思政与成果推广"
	p.Scores = []BuilderScore{}
	for _, dim := range policy.Dimensions {
		p.Scores = append(p.Scores, BuilderScore{dim, []string{"s1"}, []string{"待补：课程过程证据"}})
	}
	p.References = []BuilderReference{{"builtin:新文科:report", "借鉴本赛道评分对照", ""}}
	if _, e = validateBuilderPlan(fixtureJSON(p), "architect", refs, policy); e != nil {
		t.Fatal(e, evidence)
	}
	p.Sections[1].Title = "AI伦理与治理"
	if _, e = validateBuilderPlan(fixtureJSON(p), "architect", refs, policy); e == nil {
		t.Fatal("AI-only chapter accepted for new liberal arts")
	}
	if len(builtinRuleSearch(context.Background(), "新文科 第七届 AI工具报告评分")) == 0 {
		t.Fatal("non-AI track rules lost when mentioning AI tools")
	}
	for _, mutate := range []func(*BuilderPlan){func(p *BuilderPlan) { p.Demo = nil }, func(p *BuilderPlan) { p.DemoDuration = 16 }, func(p *BuilderPlan) { p.Demo[2].Phase = "课前" }, func(p *BuilderPlan) { p.Demo[1].ClassID = "missing" }, func(p *BuilderPlan) { p.Annotations = p.Annotations[:1] }} {
		p := builderFixture("video-planner")
		mutate(&p)
		refs := map[string]map[string]any{"builtin:rules7:scoring": {"authority_status": rules7Notice}}
		if _, e := validateBuilderPlan(fixtureJSON(p), "video-planner", refs); e == nil {
			t.Fatal("incomplete second video accepted")
		}
	}
}

// TestBuilderAuditRetractionRegressions covers the five retractions in
// research/给Codex的撤回与修正指令_事实性断言核查.md (independent audit 2026-11-17):
// ethics landing instead of mandatory independent chapter; 40–50 minute lessons;
// 课前/课后 required with 课中衔接 optional; annotations only for AI segments.
func TestBuilderAuditRetractionRegressions(t *testing.T) {
	available := map[string]map[string]any{"builtin:rules7:scoring": {"authority_status": rules7Notice}}

	// AI track: ethics landing inside another chapter, no independent chapter → pass.
	p := builderFixture("architect")
	p.Sections[1].Title = "课程思政与成果推广"
	p.Sections[1].Questions = []string{"如何引导学生负责任地使用AI并留痕"}
	if _, e := validateBuilderPlan(fixtureJSON(p), "architect", available); e != nil {
		t.Fatal("ethics landing without independent chapter rejected:", e)
	}
	// AI track without any ethics landing → reject.
	p = builderFixture("architect")
	p.Sections[1].Title = "普通建议"
	p.Sections[1].Questions = []string{"如何组织课堂"}
	if _, e := validateBuilderPlan(fixtureJSON(p), "architect", available); e == nil {
		t.Fatal("AI plan without ethics landing accepted")
	}

	// 40/50 minute lessons pass; 39/51 rejected.
	for _, dur := range []int{40, 50} {
		p = builderFixture("video-planner")
		p.Duration = dur
		p.Timeline[1].End = dur
		if _, e := validateBuilderPlan(fixtureJSON(p), "video-planner", available); e != nil {
			t.Fatalf("%d-minute lesson rejected: %v", dur, e)
		}
	}
	for _, dur := range []int{39, 51} {
		p = builderFixture("video-planner")
		p.Duration = dur
		p.Timeline[1].End = dur
		if _, e := validateBuilderPlan(fixtureJSON(p), "video-planner", available); e == nil {
			t.Fatalf("%d-minute lesson accepted", dur)
		}
	}

	// Demo without 课中衔接 passes; missing 课前 or 课后 rejected.
	p = builderFixture("video-planner")
	p.Demo = []BuilderDemo{{"d1", 0, 7, "课前", "拟介绍课前问题诊断", "拟展示AI追问", "拟展示问题清单", "t1"}, {"d2", 7, 15, "课后", "拟介绍课后延伸", "拟展示AI反馈", "拟展示修订作品", "t2"}}
	if _, e := validateBuilderPlan(fixtureJSON(p), "video-planner", available); e != nil {
		t.Fatal("two-phase demo rejected:", e)
	}
	for _, phase := range []string{"课前", "课后"} {
		p = builderFixture("video-planner")
		p.Demo = []BuilderDemo{{"d1", 0, 7, phase, "拟介绍环节设计", "拟展示AI动作", "拟展示学生产物", "t1"}, {"d2", 7, 15, "课中衔接", "拟介绍课堂衔接", "拟展示核验记录", "拟展示课堂材料", "t2"}}
		if _, e := validateBuilderPlan(fixtureJSON(p), "video-planner", available); e == nil {
			t.Fatalf("demo missing %s accepted", phase)
		}
	}

	// Segment without AI action needs no annotation; AI segment missing annotation rejected.
	p = builderFixture("video-planner")
	p.Timeline[1].AI = ""
	p.Annotations = p.Annotations[:1]
	if _, e := validateBuilderPlan(fixtureJSON(p), "video-planner", available); e != nil {
		t.Fatal("no-AI segment without annotation rejected:", e)
	}
	p = builderFixture("video-planner")
	p.Annotations = p.Annotations[:1]
	if _, e := validateBuilderPlan(fixtureJSON(p), "video-planner", available); e == nil {
		t.Fatal("AI segment missing annotation accepted")
	}

	// Prompts carry the corrected specs and case-verification discipline.
	prompt := generationPrompt("video-planner", 7)
	for _, s := range []string{"40–50", "课前、课后", "课中衔接段，可选", "有AI介入", "unverified", "transcribed"} {
		if !strings.Contains(prompt, s) {
			t.Errorf("prompt missing %q", s)
		}
	}
	for _, s := range []string{"从0到45", "三类phase", "课堂每段提供"} {
		if strings.Contains(prompt, s) {
			t.Errorf("prompt still contains retracted rule %q", s)
		}
	}
	if strings.Contains(generationPrompt("architect", 7), "独立设置AI伦理与治理章节") {
		t.Error("architect prompt still forces independent ethics chapter")
	}
}

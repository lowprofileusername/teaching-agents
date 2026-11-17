package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func isBuilder(id string) bool {
	return id == "architect" || id == "video-planner" || id == "presentation-architect" || id == "syllabus-architect"
}
func builderEdition(c Chat) int {
	if n := queryEdition(strings.Split(c.Question, "\n本轮备赛信息")[0]); n != 0 {
		return n
	}
	if c.Edition == "6" {
		return 6
	}
	return 7
}

type BuilderSection struct {
	Draft      string   `json:"draft,omitempty"`
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	WordBudget int      `json:"word_budget"`
	Questions  []string `json:"questions"`
	Evidence   []string `json:"evidence_needed"`
}
type BuilderScore struct {
	Dimension string   `json:"dimension"`
	Sections  []string `json:"section_ids"`
	Evidence  []string `json:"evidence_needed"`
}
type BuilderStage struct {
	ID      string `json:"id"`
	Start   int    `json:"start_minute"`
	End     int    `json:"end_minute"`
	Stage   string `json:"stage"`
	AI      string `json:"ai_action"`
	Student string `json:"student_evidence"`
	Measure string `json:"report_measure"`
}
type BuilderReference struct {
	ID     string `json:"source_id"`
	Use    string `json:"use"`
	Notice string `json:"notice"`
}
type BuilderDemo struct {
	ID        string `json:"id"`
	Start     int    `json:"start_minute"`
	End       int    `json:"end_minute"`
	Phase     string `json:"phase"`
	Narration string `json:"narration"`
	AI        string `json:"ai_action"`
	Evidence  string `json:"student_evidence"`
	ClassID   string `json:"class_segment_id"`
}
type BuilderAnnotation struct {
	SegmentID string `json:"segment_id"`
	Purpose   string `json:"purpose"`
	Operation string `json:"operation"`
	Expected  string `json:"expected_effect"`
}
type BuilderPlan struct {
	Slides       []BuilderSlide      `json:"slides,omitempty"`
	Syllabus     *BuilderSyllabus    `json:"syllabus,omitempty"`
	Warnings     []string            `json:"warnings,omitempty"`
	DemoDuration int                 `json:"demo_duration_minutes"`
	Demo         []BuilderDemo       `json:"demo_script"`
	Annotations  []BuilderAnnotation `json:"lesson_annotations"`

	Kind       string             `json:"kind"`
	Line       string             `json:"innovation_line"`
	Loop       string             `json:"logic_loop"`
	WordBudget int                `json:"word_budget_total"`
	Duration   int                `json:"duration_minutes"`
	Sections   []BuilderSection   `json:"sections"`
	Scores     []BuilderScore     `json:"scoring_map"`
	Evidence   []string           `json:"evidence_needed"`
	Timeline   []BuilderStage     `json:"timeline"`
	References []BuilderReference `json:"references"`
	Boundary   string             `json:"boundary"`
}

var generatedFactRisk = regexp.MustCompile(`[0-9０-９%％“”「」]|百分之|[一二三四五六七八九十百千万]+(名学生|人次|人)|学生(表示|说[：:“”「」]|反馈称)|已(提升|提高|获|被|实现|采用)|获得[^。！？\n]*奖|被[^。！？\n]*(采纳|采用)`)

// hasAIAction reports whether a timeline segment describes an AI intervention.
// Empty or explicitly "no AI" segments do not require lesson-plan annotations.
func hasAIAction(s string) bool {
	s = strings.TrimSpace(s)
	return s != "" && s != "无" && s != "无AI动作" && s != "本环节无AI"
}

func validateBuilderPlan(raw, role string, available map[string]map[string]any, policies ...BuilderPolicy) (BuilderPlan, error) {
	policy, _ := builderPolicyFor("人工智能", 7)
	if len(policies) > 0 {
		policy = policies[0]
	}
	var p BuilderPlan
	// Reject unknown result/metrics fields instead of silently accepting them.
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if e := dec.Decode(&p); e != nil {
		return p, fmt.Errorf("生成JSON格式不完整：%w", e)
	}
	if e := dec.Decode(new(any)); e != io.EOF {
		return p, fmt.Errorf("生成结果含额外内容")
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal([]byte(raw), &fields)
	for _, key := range []string{"kind", "innovation_line", "logic_loop", "word_budget_total", "duration_minutes", "sections", "scoring_map", "evidence_needed", "timeline", "references", "boundary", "demo_duration_minutes", "demo_script", "lesson_annotations"} {
		if len(fields[key]) == 0 || string(fields[key]) == "null" {
			return p, fmt.Errorf("生成结果缺少字段%s", key)
		}
	}
	for key, required := range map[string][]string{
		"demo_script":        {"id", "start_minute", "end_minute", "phase", "narration", "ai_action", "student_evidence", "class_segment_id"},
		"lesson_annotations": {"segment_id", "purpose", "operation", "expected_effect"},
		"sections":           {"id", "title", "word_budget", "questions", "evidence_needed"},
		"timeline":           {"id", "start_minute", "end_minute", "stage", "ai_action", "student_evidence", "report_measure"},
		"scoring_map":        {"dimension", "section_ids", "evidence_needed"},
		"references":         {"source_id", "use", "notice"},
	} {
		var rows []map[string]json.RawMessage
		if json.Unmarshal(fields[key], &rows) != nil {
			return p, fmt.Errorf("字段%s需为数组", key)
		}
		for _, row := range rows {
			for _, name := range required {
				if len(row[name]) == 0 || string(row[name]) == "null" {
					return p, fmt.Errorf("%s缺少字段%s", key, name)
				}
			}
		}
	}
	texts := []string{p.Line, p.Loop}
	ids := map[string]bool{}
	total := 0
	ethicsChapter := false
	ethicsLanding := false
	aiSegments := map[string]bool{}
	evidenceOK := func(a []string) bool {
		if len(a) == 0 {
			return false
		}
		for _, s := range a {
			if !strings.HasPrefix(s, "待补：") || len([]rune(s)) < 5 {
				return false
			}
			texts = append(texts, s)
		}
		return true
	}
	for _, s := range p.Sections {
		if s.ID == "" || ids[s.ID] || s.WordBudget <= 0 || strings.TrimSpace(s.Title) == "" || len(s.Questions) == 0 || !evidenceOK(s.Evidence) {
			return p, fmt.Errorf("章节预算、问题或待补证据不完整")
		}
		ids[s.ID] = true
		total += s.WordBudget
		texts = append(texts, s.Title)
		texts = append(texts, s.Questions...)
		if strings.Contains(strings.ReplaceAll(s.Title, " ", ""), "AI伦理与治理") {
			ethicsChapter = true
		}
		for _, q := range append([]string{s.Title}, s.Questions...) {
			if strings.Contains(q, "伦理") || strings.Contains(q, "治理") || strings.Contains(q, "负责任") {
				ethicsLanding = true
			}
		}
	}
	end := 0
	for _, t := range p.Timeline {
		if t.ID == "" || ids[t.ID] || t.Start != end || t.End <= t.Start || t.End > 50 {
			return p, fmt.Errorf("课堂时刻表必须连续覆盖40–50分钟的完整1学时")
		}
		end = t.End
		ids[t.ID] = true
		aiSegments[t.ID] = hasAIAction(t.AI)
		texts = append(texts, t.Stage, t.Student, t.Measure)
		if strings.TrimSpace(t.AI) != "" {
			texts = append(texts, t.AI)
		}
	}
	if role == "architect" {
		if p.Kind != "report" || p.WordBudget != policy.Budget || total != policy.Budget || p.Duration != 0 || len(p.Timeline) != 0 || p.DemoDuration != 0 || len(p.Demo) != 0 || len(p.Annotations) != 0 {
			return p, fmt.Errorf("报告预算、章节或结构不符")
		}
		// 官方评分原文无"独立伦理章"要求：AI赛道只要求伦理治理内容有落点（可在课程思政章节内）。
		if policy.AI && !ethicsLanding {
			return p, fmt.Errorf("AI赛道伦理治理/负责任使用AI内容须有落点（可置于课程思政章节，不要求独立成章）")
		}
		if !policy.AI && ethicsChapter {
			return p, fmt.Errorf("非AI赛道不得设置AI专属伦理治理章节")
		}
	} else if role == "video-planner" {
		if p.Kind != "lesson" || p.Duration < 40 || p.Duration > 50 || end != p.Duration || p.WordBudget != 0 || len(p.Sections) != 0 {
			return p, fmt.Errorf("视频策划需40–50分钟完整课堂时刻表")
		}
	}
	if role == "video-planner" {
		if !policy.AI {
			return p, fmt.Errorf("双视频策划适用于人工智能赛道，请选对应赛道")
		}
		if p.DemoDuration <= 0 || p.DemoDuration > 15 || len(p.Demo) < 2 {
			return p, fmt.Errorf("缺少不超过15分钟的课外说课脚本")
		}
		phases := map[string]bool{"课前": false, "课中衔接": false, "课后": false}
		demoIDs := map[string]bool{}
		demoEnd := 0
		for _, d := range p.Demo {
			if d.ID == "" || demoIDs[d.ID] || !ids[d.ClassID] || d.Start != demoEnd || d.End <= d.Start || d.End > p.DemoDuration {
				return p, fmt.Errorf("说课时间或课堂关联无效")
			}
			if _, ok := phases[d.Phase]; !ok {
				return p, fmt.Errorf("说课需标明课前、课中衔接或课后")
			}
			phases[d.Phase] = true
			demoIDs[d.ID] = true
			demoEnd = d.End
			texts = append(texts, d.Narration, d.AI, d.Evidence)
		}
		// 官方只要求说课展示课前、课后环节；课中衔接为可选写作建议，不作拒绝条件。
		if demoEnd != p.DemoDuration || !phases["课前"] || !phases["课后"] {
			return p, fmt.Errorf("说课必须覆盖课前、课后（课中衔接可选），连续完整")
		}
		annotations := map[string]bool{}
		for _, a := range p.Annotations {
			if !ids[a.SegmentID] || annotations[a.SegmentID] {
				return p, fmt.Errorf("教案标注关联错误")
			}
			annotations[a.SegmentID] = true
			texts = append(texts, a.Purpose, a.Operation, a.Expected)
		}
		// 官方要求教案课件标注AI应用环节四项；只覆盖有AI介入的环节，无AI动作环节不强制。
		for id, need := range aiSegments {
			if need && !annotations[id] {
				return p, fmt.Errorf("有AI介入的课堂环节需补四项标注（应用环节/设计目的/操作流程/预期效果）")
			}
		}
	}
	if role == "presentation-architect" {
		if e := validatePresentation(&p, available, ids); e != nil {
			return p, e
		}
	} else if len(p.Slides) > 0 {
		return p, fmt.Errorf("当前角色不能混入PPT方案")
	}
	if role == "syllabus-architect" {
		if e := validateSyllabus(&p, policy.AI, ids); e != nil {
			return p, e
		}
	} else if p.Syllabus != nil {
		return p, fmt.Errorf("当前角色不能混入大纲方案")
	}
	if e := validateFullDraft(p, policy.Depth); e != nil {
		return p, e
	}
	dims := map[string]bool{}
	for _, d := range policy.Dimensions {
		dims[d] = false
	}
	if len(p.Scores) != len(policy.Dimensions) {
		return p, fmt.Errorf("需要覆盖当前赛道全部评分维度")
	}
	for _, s := range p.Scores {
		used, ok := dims[s.Dimension]
		if !ok || used || len(s.Sections) == 0 || !evidenceOK(s.Evidence) {
			return p, fmt.Errorf("评分维度、对应位置或证据不完整")
		}
		dims[s.Dimension] = true
		for _, id := range s.Sections {
			if !ids[id] {
				return p, fmt.Errorf("评分落点引用不存在的章节或环节")
			}
		}
	}
	if !evidenceOK(p.Evidence) || len(p.References) == 0 {
		return p, fmt.Errorf("缺少待补证据或实际来源")
	}
	for i, ref := range p.References {
		src, ok := available[ref.ID]
		if !ok {
			return p, fmt.Errorf("引用了本轮未提供的来源")
		}
		texts = append(texts, ref.Use)
		p.References[i].Notice = fmt.Sprint(src["authority_status"]) + " 来源条目：" + fmt.Sprint(src["title"])
		if origin := fmt.Sprint(src["origin"]); origin != "" && origin != "<nil>" {
			p.References[i].Notice += " 来源专家：" + origin
		}
	}
	for _, s := range texts {
		if strings.TrimSpace(s) == "" || generatedFactRisk.MatchString(s) {
			return p, fmt.Errorf("生成文本含空字段、统计数字、引语或成效断言；未计为完成，请只列结构与待补证据")
		}
	}
	if e := validateDeepCitations(p, available, policy.AI); e != nil {
		return p, e
	}
	p.Boundary = policy.boundary()
	return p, nil
}
func aiReportDimensions() []string {
	var r struct {
		Scoring struct {
			Dimensions []struct {
				Name string `json:"name"`
			} `json:"report_dimensions_ai_track"`
		} `json:"scoring"`
	}
	if readReferenceJSON("knowledge/rules_7_ai_2026.json", &r) != nil {
		return nil
	}
	out := []string{}
	for _, d := range r.Scoring.Dimensions {
		out = append(out, d.Name)
	}
	return out
}
func appendBuiltinEvidence(ctx context.Context, c Chat) string {
	q := c.Track + " " + strings.Split(c.Question, "\n本轮备赛信息")[0]
	if queryEdition(q) == 0 {
		if c.Edition == "6" {
			q = "第六届 " + q
		} else if c.Edition == "7" || c.Mode == "generate" {
			q = "第七届 " + q
		}
	}
	if c.Mode == "generate" {
		if p, e := builderPolicyFor(c.Track, builderEdition(c)); e == nil && !p.AI {
			q += " " + p.Track + " 报告骨架"
		} else {
			q += " 人工智能 报告骨架 视频策划 打法 案例"
		}
	}
	found := builtinSearch(ctx, q)
	if c.Mode == "generate" {
		policy, _ := builderPolicyFor(c.Track, builderEdition(c))
		selected := []map[string]any{}
		if policy.AI {
			selected = builderDeepEvidence(ctx, q)
		}
		counts := map[string]int{}
		for _, r := range found {
			if duplicateDeepPlay(stringField(r, "id"), selected) {
				continue
			}
			kind := stringField(r, "kind")
			if strings.HasPrefix(stringField(r, "id"), "builtin:aideep:") {
				continue
			}
			if !policy.AI && kind != "competition_rule" && kind != "track_migration" {
				continue
			}
			limit := 0
			switch kind {
			case "competition_rule":
				if (builderEdition(c) == 6 || c.Track != "人工智能" && c.Track != "" && c.Track != "未选赛道") && strings.HasSuffix(stringField(r, "id"), ":report") {
					limit = 1
				}
			case "provisional_rule":
				if r["id"] == "builtin:rules7:scoring" || r["id"] == "builtin:rules7:materials" {
					limit = 1
				}
			case "track_migration":
				if stringField(r, "track") == policy.Track {
					limit = 2
				}
			case "ai_migration":
				limit = 2
			case "award_case":
				limit = 3
			}
			if counts[kind] < limit {
				selected = append(selected, r)
				counts[kind]++
			}
		}
		if policy.AI && counts["ai_migration"] == 0 {
			// Deep sessions must not starve distinct cross-session mechanisms.
			for _, id := range []string{"builtin:play:ai_refuse_direct_answer", "builtin:play:ai_blind_review"} {
				if r := builtinReferenceByID(ctx, id); r != nil {
					selected = append(selected, r)
				}
			}
		}
		found = selected
	}
	if len(found) == 0 {
		return c.Evidence
	}
	root := map[string]any{}
	if c.Evidence != "" && json.Unmarshal([]byte(c.Evidence), &root) != nil {
		root = map[string]any{"provided_reference_unverified": c.Evidence}
	}
	if root == nil {
		root = map[string]any{}
	}
	seen := map[string]bool{}
	existing := asList(root["document_evidence"])
	if c.Mode == "generate" {
		kept := []any{}
		for _, v := range existing {
			if m, ok := v.(map[string]any); ok && !strings.HasPrefix(stringField(m, "id"), "builtin:") {
				kept = append(kept, v)
			}
		}
		existing = kept
	}
	for _, x := range existing {
		if m, ok := x.(map[string]any); ok {
			seen[fmt.Sprint(m["id"])] = true
		}
	}
	for _, x := range found {
		if !seen[fmt.Sprint(x["id"])] {
			existing = append(existing, x)
		}
	}
	root["document_evidence"] = existing
	b, _ := json.Marshal(root)
	return string(b)
}
func includedBuilderReferences(raw string) map[string]map[string]any {
	out := map[string]map[string]any{}
	var root map[string]any
	_ = json.Unmarshal([]byte(raw), &root)
	for _, key := range []string{"document_evidence", "sources"} {
		for _, r := range objects(root[key]) {
			id := stringField(r, "id")
			if strings.HasPrefix(id, "builtin:") {
				out[id] = r
			}
		}
	}
	return out
}

// Buffer and validate builder output before publishing, including stream requests.
func (a *App) completeGeneration(w http.ResponseWriter, r *http.Request, cfg Config, payload map[string]any, meta map[string]any, c Chat, packed string) {
	start := time.Now()
	payload["stream"] = false
	b, _ := json.Marshal(payload)
	data, e := a.post(r.Context(), cfg.Base+"/chat/completions", cfg.Key, "Authorization", "application/json", bytes.NewReader(b))
	if e != nil {
		fail(w, 502, e.Error())
		return
	}
	var out modelResponse
	if json.Unmarshal(data, &out) != nil || len(out.Choices) == 0 {
		fail(w, 502, "生成接口未返回有效结构")
		return
	}
	ch := out.Choices[0]
	if ch.FinishReason == "length" {
		fail(w, 502, "生成输出截断，未计为完成；请增加最大输出后重试")
		return
	}
	answer, reason := responseText(ch.Message.Content, reasoningOf(ch.Message), true)
	policy, e := builderPolicyFor(c.Track, builderEdition(c))
	if e != nil {
		fail(w, 400, e.Error())
		return
	}
	policy.Depth = c.Depth
	p, e := validateBuilderPlan(answer, c.SpecialistID, includedBuilderReferences(packed), policy)
	if e != nil {
		fail(w, 502, e.Error())
		return
	}
	p.Boundary = policy.boundary()
	encoded, _ := json.Marshal(p)
	meta["mode"] = "generate"
	meta["generation_validated"] = true
	result := chatResult(string(encoded), reason, ch.FinishReason, meta, time.Since(start))
	result["plan"] = p
	result["plan_requirements"] = map[string]any{"track": policy.Track, "notice": policy.Notice, "report": policy.Report, "video": policy.Video}
	jsonOut(w, 200, result)
}

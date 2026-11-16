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

func isBuilder(id string) bool { return id == "architect" || id == "video-planner" }
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
	ethics := false
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
			ethics = true
		}
	}
	end := 0
	for _, t := range p.Timeline {
		if t.ID == "" || ids[t.ID] || t.Start != end || t.End <= t.Start || t.End > 45 {
			return p, fmt.Errorf("课堂时刻表必须连续覆盖完整45分钟")
		}
		end = t.End
		ids[t.ID] = true
		texts = append(texts, t.Stage, t.AI, t.Student, t.Measure)
	}
	if role == "architect" {
		if p.Kind != "report" || p.WordBudget != policy.Budget || total != policy.Budget || (policy.AI && !ethics) || (!policy.AI && ethics) || p.Duration != 0 || len(p.Timeline) != 0 || p.DemoDuration != 0 || len(p.Demo) != 0 || len(p.Annotations) != 0 {
			return p, fmt.Errorf("报告预算、章节或赛道要求不符（AI赛道须独立伦理治理章）")
		}
	} else {
		if p.Kind != "lesson" || p.Duration != 45 || end != 45 || p.WordBudget != 0 || len(p.Sections) != 0 {
			return p, fmt.Errorf("视频策划需完整45分钟时刻表")
		}
	}
	if role == "video-planner" {
		if !policy.AI {
			return p, fmt.Errorf("双视频策划适用于人工智能赛道，请选对应赛道")
		}
		if p.DemoDuration <= 0 || p.DemoDuration > 15 || len(p.Demo) < 3 {
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
		if demoEnd != p.DemoDuration || !phases["课前"] || !phases["课中衔接"] || !phases["课后"] {
			return p, fmt.Errorf("说课必须覆盖课前课中衔接课后，连续完整")
		}
		annotations := map[string]bool{}
		for _, a := range p.Annotations {
			if !ids[a.SegmentID] || annotations[a.SegmentID] {
				return p, fmt.Errorf("教案标注关联错误")
			}
			annotations[a.SegmentID] = true
			texts = append(texts, a.Purpose, a.Operation, a.Expected)
		}
		if len(annotations) != len(p.Timeline) {
			return p, fmt.Errorf("每个课堂环节需标注AI应用、设计目的、操作流程、预期效果")
		}
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
		counts := map[string]int{}
		for _, r := range found {
			kind := stringField(r, "kind")
			if !policy.AI && kind != "competition_rule" {
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

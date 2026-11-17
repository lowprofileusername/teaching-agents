package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

type BuilderSlide struct {
	ID             string   `json:"id"`
	PageNo         int      `json:"page_no"`
	Title          string   `json:"title"`
	Section        string   `json:"section"`
	Bullets        []string `json:"bullets"`
	Visual         string   `json:"visual"`
	EvidenceSource string   `json:"evidence_source"`
	SpeakerNote    string   `json:"speaker_note"`
	DurationSec    int      `json:"duration_sec"`
}

var presentationIdentity = regexp.MustCompile(`大学|学院|学校|院系|姓名[：:]|汇报人[：:]|授课教师[：:]`)

func anonymousPresentation(text string) bool {
	if presentationIdentity.MatchString(text) {
		return false
	}
	var catalog map[string]any
	if readReferenceJSON("knowledge/experts.json", &catalog) != nil {
		return false
	}
	for _, x := range objects(catalog["experts"]) {
		if name := stringField(x, "name"); name != "" && strings.Contains(text, name) {
			return false
		}
	}
	return true
}
func validatePresentation(p *BuilderPlan, available map[string]map[string]any, ids map[string]bool) error {
	if p.Kind != "presentation" || len(p.Slides) == 0 || len(p.Sections) != 0 || p.WordBudget != 0 || p.Duration != 0 || len(p.Timeline) != 0 || len(p.Demo) != 0 || p.DemoDuration != 0 || len(p.Annotations) != 0 {
		return fmt.Errorf("PPT方案需非空slides，报告/课堂专属字段留空或零")
	}
	total := 0
	groups := map[string]bool{}
	p.Warnings = nil
	for i, s := range p.Slides {
		if s.ID == "" || ids[s.ID] || s.PageNo != i+1 || s.DurationSec <= 0 || len(s.Bullets) == 0 || len(s.Bullets) > 4 {
			return fmt.Errorf("PPT页序、ID、时间或要点不完整（每页至多四条）")
		}
		for _, v := range []string{s.Title, s.Section, s.Visual, s.SpeakerNote} {
			if strings.TrimSpace(v) == "" {
				return fmt.Errorf("PPT必填字段为空")
			}
		}
		if available[s.EvidenceSource] == nil {
			return fmt.Errorf("PPT证据来源不在本轮实际提供的资料内")
		}
		body := strings.Join(append([]string{s.Title, s.Section, s.Visual, s.SpeakerNote}, s.Bullets...), "\n")
		if !anonymousPresentation(body) {
			return fmt.Errorf("PPT内容须匿名，不能出现姓名、学校或院系；请用匿名占位描述")
		}
		if generatedFactRisk.MatchString(body) {
			return fmt.Errorf("PPT含统计数字、引语或成效断言；请使用待补证据占位")
		}
		for _, b := range s.Bullets {
			if strings.TrimSpace(b) == "" {
				return fmt.Errorf("PPT要点为空")
			}
		}
		ids[s.ID] = true
		total += s.DurationSec
		groups[s.Section] = true
		// Ensure every page's source receives server-owned provenance in the UI.
		seen := false
		for _, r := range p.References {
			if r.ID == s.EvidenceSource {
				seen = true
			}
		}
		if !seen {
			p.References = append(p.References, BuilderReference{s.EvidenceSource, "用作页面结构参考，用户成效仍待补", ""})
		}
	}
	if total > 720 {
		return fmt.Errorf("汇报总时长%d秒，超过官方十二分钟上限（720秒）", total)
	}
	if len(p.Slides) > 30 {
		p.Warnings = append(p.Warnings, "页数超过30页：众师云12923的第三方建议，非官方硬限制。")
	}
	if !groups["课程与痛点"] || !groups["创新内容"] || !groups["成效与推广"] {
		p.Warnings = append(p.Warnings, "未使用3+6+3三段结构：这是AI综合建议，无官方或具名专家原文，不阻断方案。")
	}
	if total < 690 || total > 710 {
		p.Warnings = append(p.Warnings, "总时长不在690–710秒的AI建议目标内；只要不超过720秒即可通过时长校验。")
	}
	return nil
}
func validateFullDraft(p BuilderPlan, depth string) error {
	total := 0
	for _, s := range p.Sections {
		if depth == "full" && strings.TrimSpace(s.Draft) == "" {
			return fmt.Errorf("逐段扩写缺少正文draft，未计完成")
		}
		if s.Draft != "" {
			if generatedFactRisk.MatchString(s.Draft) {
				return fmt.Errorf("扩写草稿含具体成效数字、引语或获奖断言；必须用待补占位")
			}
			n := utf8.RuneCountInString(s.Draft)
			total += n
			if n > s.WordBudget {
				return fmt.Errorf("草稿章节超出本节规划预算，请重新生成；未自动删减正文")
			}
		}
	}
	if total > 4000 {
		return fmt.Errorf("扩写草稿超出四千字规划预算；未自动删减正文")
	}
	return nil
}

const fullDraftPrompt = `本轮depth=full：每个sections对象必须带draft正文。按痛点证据→改革思路→具体做法→可补证据位逐段成文。所有成效数字、学生原话、获奖信息只能保留待补：XXX占位，不填任何具体数字或引语。正文使用拟议措辞，未提供事实不得写已实施。每节正文字符数不超过本节word_budget，合计不超过四千字；不要为了字数自动删用户正文。AI伦理治理在相关章节有落点即可，不强制独立章；非AI赛道依本赛道问题展开。`
const presentationPrompt = `你是教学创新汇报PPT架构师，只规划可编辑的页序/要点/讲稿方案，不冒称已生成PPTX。本轮kind=presentation。sections/timeline/demo_script/lesson_annotations为空数组，word_budget_total/duration_minutes/demo_duration_minutes为零。另加slides非空数组，每项字段：{"id":"p1","page_no":1,"title":"观点式标题","section":"课程与痛点","bullets":["待补：真实问题证据"],"visual":"按内容选择所需图表或截图并匿名化，不预设固定图样","evidence_source":"本轮实际source id","speaker_note":"拟议讲稿要点，事实缺失用待补","duration_sec":30}。id唯一、page_no从一连续，每页一观点、bullets一到四条，scoring_map的section_ids引用slides.id。所有页面文字、讲稿、截图占位不得包含姓名/学校/院系；来源专家只放references，不放幻灯片。每页source只能从本轮available选择。总时间不超过720秒是硬性上限；不把30页、3+6+3、690–710秒当硬限制。30页来自众师云12923第三方建议；3+6+3（课程与痛点/创新内容/成效与推广）与690–710秒均为AI综合建议，无具名专家原文。数据仍用待补占位，不填写学生人数、分数、提升率、引语、获奖或采用。`

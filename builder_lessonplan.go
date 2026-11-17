package main

import (
	"fmt"
	"strings"
)

// LessonPlanStage is one minute-level teaching stage of the recorded segment.
// When AI is involved, the four official annotation fields are required:
// application stage / design purpose / operation flow / expected effect.
type LessonPlanStage struct {
	Stage         string `json:"stage"`
	Minutes       string `json:"minutes"`
	Teacher       string `json:"teacher"`
	Student       string `json:"student"`
	AI            string `json:"ai_action,omitempty"`
	Intent        string `json:"intent"`
	AIApplication string `json:"ai_application,omitempty"`
	AIPurpose     string `json:"ai_purpose,omitempty"`
	AIOperation   string `json:"ai_operation,omitempty"`
	AIExpected    string `json:"ai_expected,omitempty"`
}

// BuilderLessonPlan is the segment-level teaching design (教案) accompanying
// the recorded video segment. Structure follows the scoring dimensions
// (objectives / learner analysis / key-difficult points / process / assessment
// / design innovation); the AI four-item annotation follows the official AI
// track notice clause.
type BuilderLessonPlan struct {
	CourseName       string            `json:"course_name"`
	SegmentTopic     string            `json:"segment_topic"`
	SegmentHours     string            `json:"segment_hours"`
	Audience         string            `json:"audience"`
	Objectives       []string          `json:"objectives"`
	LearnerAnalysis  string            `json:"learner_analysis"`
	KeyPoints        string            `json:"key_points"`
	Difficulties     string            `json:"difficulties"`
	Stages           []LessonPlanStage `json:"stages"`
	Assessment       string            `json:"assessment"`
	Resources        string            `json:"resources"`
	IdeologyPolitics string            `json:"ideology_politics"`
	Consistency      string            `json:"consistency"`
}

var lessonPlanPartIDs = []string{"lessonplan:objectives", "lessonplan:learner", "lessonplan:keydiff", "lessonplan:process", "lessonplan:assessment", "lessonplan:ideology", "lessonplan:ai"}

func hasStageAI(s LessonPlanStage) bool {
	t := strings.TrimSpace(s.AI)
	return t != "" && t != "无" && t != "无AI动作" && t != "本环节无AI"
}

func validateLessonPlan(p *BuilderPlan, ai bool, ids map[string]bool) error {
	if p.Kind != "lessonplan" || p.LessonPlan == nil || len(p.Sections) != 0 || p.WordBudget != 0 || p.Duration != 0 || len(p.Timeline) != 0 || len(p.Demo) != 0 || p.DemoDuration != 0 || len(p.Annotations) != 0 || len(p.Slides) != 0 || p.Syllabus != nil {
		return fmt.Errorf("教案方案需非空lesson_plan对象，报告/课堂/PPT/大纲专属字段留空或零")
	}
	l := p.LessonPlan
	for _, v := range []string{l.CourseName, l.SegmentTopic, l.SegmentHours, l.Audience, l.LearnerAnalysis, l.KeyPoints, l.Difficulties, l.Assessment, l.Resources, l.IdeologyPolitics, l.Consistency} {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("教案必备要素为空（课程/节段/学时/对象/学情/重难点/评价/资源/思政/一致性）")
		}
	}
	if len(l.Objectives) == 0 || len(l.Stages) == 0 {
		return fmt.Errorf("教案节段目标与教学过程不能为空")
	}
	for _, o := range l.Objectives {
		if strings.TrimSpace(o) == "" {
			return fmt.Errorf("教案节段目标存在空项")
		}
	}
	hasAI := false
	for _, s := range l.Stages {
		if strings.TrimSpace(s.Stage) == "" || strings.TrimSpace(s.Minutes) == "" || strings.TrimSpace(s.Teacher) == "" || strings.TrimSpace(s.Student) == "" || strings.TrimSpace(s.Intent) == "" {
			return fmt.Errorf("教案过程每环节须含环节名/分钟/教师活动/学生活动/设计意图")
		}
		if hasStageAI(s) {
			hasAI = true
			// 官方AI赛道通知：教案课件须标注应用环节/设计目的/操作流程/预期效果四项。
			for _, v := range []string{s.AIApplication, s.AIPurpose, s.AIOperation, s.AIExpected} {
				if strings.TrimSpace(v) == "" {
					return fmt.Errorf("有AI介入的环节须补齐四项标注（应用环节/设计目的/操作流程/预期效果）")
				}
			}
		}
	}
	if ai && !hasAI {
		return fmt.Errorf("AI赛道教案须至少一个环节有AI介入并标注四项")
	}
	body := strings.Join([]string{l.CourseName, l.SegmentTopic, l.SegmentHours, l.Audience, l.LearnerAnalysis, l.KeyPoints, l.Difficulties, l.Assessment, l.Resources, l.IdeologyPolitics, l.Consistency}, "\n")
	body += "\n" + strings.Join(l.Objectives, "\n")
	for _, s := range l.Stages {
		body += "\n" + strings.Join([]string{s.Stage, s.Minutes, s.Teacher, s.Student, s.AI, s.Intent, s.AIApplication, s.AIPurpose, s.AIOperation, s.AIExpected}, " ")
	}
	if syllabusFactRisk.MatchString(body) {
		return fmt.Errorf("教案含成效断言（提升率/满意度/获奖/采用等）；用户事实用待补占位，分钟分配等结构数据除外")
	}
	if !anonymousPresentation(body) {
		return fmt.Errorf("竞赛版教案须匿名：不得出现姓名/学校/院系")
	}
	for _, id := range lessonPlanPartIDs {
		ids[id] = true
	}
	p.Warnings = nil
	if strings.Contains(l.SegmentHours, "待补") {
		p.Warnings = append(p.Warnings, "节段学时为待补占位：课堂实录须为完整1学时40–50分钟，请按实际核实。")
	}
	if ai {
		obj := strings.Join(l.Objectives, " ")
		if !strings.Contains(obj, "数字素养") && !strings.Contains(obj, "AI") && !strings.Contains(obj, "智能") {
			p.Warnings = append(p.Warnings, "建议节段目标显式写入数字素养/AI能力/智能伦理（AI赛道评分要点），不阻断。")
		}
	}
	return nil
}

const lessonPlanPrompt = `你是教案架构师，只生成对应视频节段的节段级教学设计（教案）草案，不冒称已定稿。本轮kind=lessonplan。sections/timeline/demo_script/lesson_annotations/slides为空数组，syllabus不填，word_budget_total/duration_minutes/demo_duration_minutes为零。另加lesson_plan对象，字段：{"course_name":"课程名称","segment_topic":"对应实录节段的授课内容","segment_hours":"节段学时（完整1学时40–50分钟，未知用待补）","audience":"授课对象（人数待补）","objectives":["节段目标，知识/能力/素养，可测量"],"learner_analysis":"学情分析（已有知识/能力/习惯，真实数据待补）","key_points":"教学重点","difficulties":"教学难点","stages":[{"stage":"环节名","minutes":"分钟分配","teacher":"教师活动","student":"学生活动","ai_action":"AI活动（无则留空）","intent":"设计意图","ai_application":"AI应用环节","ai_purpose":"设计目的","ai_operation":"操作流程","ai_expected":"预期效果"}],"assessment":"节段评价（过程性+终结性，含评价任务与量规思路）","resources":"课件/平台/智能体等资源","ideology_politics":"思政融入点（从专业内容自然生长）","consistency":"与大纲/报告/实录的一致性说明"}。
依据：省赛通知配套材料明文要求"课堂教学实录视频内容对应的教案和课件"（海南科技职业大学第七届、河南/广东第七届同文）；AI赛道通知明文"在配套提交的教案与课件中，明确标注其应用环节、设计目的、操作流程及预期效果"。教案内部结构对标评分维度：目标/学情/重难点/过程与方法/考评与反馈/设计创新（第六届海南赛区评分标准、第六届国赛评分.doc）。
各赛道通用框架；人工智能赛道stages须至少一个环节有AI介入，且每个有AI介入的环节四项标注（ai_application/ai_purpose/ai_operation/ai_expected）必填；其他赛道有AI介入的环节同样补齐四项。分钟分配是结构数据，可拟议并覆盖完整节段，标待用户核实；学生人数/成绩/提升率/满意度/获奖等用户事实一律待补占位，绝不编造。竞赛版匿名，不出现姓名/学校/院系。scoring_map的section_ids用固定id：lessonplan:objectives、lessonplan:learner、lessonplan:keydiff、lessonplan:process、lessonplan:assessment、lessonplan:ideology、lessonplan:ai，恰好覆盖所给评分维度。references至少引用本轮提供的规则来源。`
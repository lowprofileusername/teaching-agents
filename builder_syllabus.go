package main

import (
	"fmt"
	"regexp"
	"strings"
)

// SyllabusUnit is one chapter/unit row of the content-and-hours schedule.
type SyllabusUnit struct {
	Unit      string `json:"unit"`
	Hours     string `json:"hours"`
	KeyPoints string `json:"key_points"`
	AIStage   string `json:"ai_stage,omitempty"`
}

// BuilderSyllabus is the competition-version course syllabus framework.
// Core elements follow the official notice list (course name / nature /
// credits-hours / audience / overview / objectives / content-schedule /
// assessment / resources); ideology-politics objectives follow the published
// competition syllabus templates; AI integration is required for the AI track.
type BuilderSyllabus struct {
	CourseName       string         `json:"course_name"`
	CourseNature     string         `json:"course_nature"`
	CreditsHours     string         `json:"credits_hours"`
	Audience         string         `json:"audience"`
	Overview         string         `json:"overview"`
	Objectives       []string       `json:"objectives"`
	IdeologyPolitics string         `json:"ideology_politics"`
	Prerequisites    string         `json:"prerequisites,omitempty"`
	ContentSchedule  []SyllabusUnit `json:"content_schedule"`
	Assessment       string         `json:"assessment"`
	Resources        []string       `json:"resources"`
	AIIntegration    string         `json:"ai_integration,omitempty"`
	Consistency      string         `json:"consistency"`
}

// syllabusFactRisk bans fabricated outcome claims while permitting structural
// course data (hours / credits / assessment weights) that a syllabus must carry.
// Unknown user facts still must be "待补：" placeholders per the prompt.
var syllabusFactRisk = regexp.MustCompile(`(满意度|及格率|优秀率|提升率)[^。！？\n]*[0-9０-９]|[0-9０-９]+[%％][^。！？\n]*(满意度|及格率|优秀率|提升|提高)|提升[0-9０-９]|学生(表示|说[：:“”「」]|反馈称)|已(提升|提高|获|被|实现|采用)|获得[^。！？\n]*奖|被[^。！？\n]*(采纳|采用)|[一二三四五六七八九十百千万]+(名学生|人次)`)

var syllabusPartIDs = []string{"syllabus:basic", "syllabus:objectives", "syllabus:ideology", "syllabus:content", "syllabus:assessment", "syllabus:resources", "syllabus:ai"}

func validateSyllabus(p *BuilderPlan, ai bool, ids map[string]bool) error {
	if p.Kind != "syllabus" || p.Syllabus == nil || len(p.Sections) != 0 || p.WordBudget != 0 || p.Duration != 0 || len(p.Timeline) != 0 || len(p.Demo) != 0 || p.DemoDuration != 0 || len(p.Annotations) != 0 || len(p.Slides) != 0 {
		return fmt.Errorf("大纲方案需非空syllabus对象，报告/课堂/PPT专属字段留空或零")
	}
	s := p.Syllabus
	for _, v := range []string{s.CourseName, s.CourseNature, s.CreditsHours, s.Audience, s.Overview, s.IdeologyPolitics, s.Assessment, s.Consistency} {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("大纲必备要素为空（名称/性质/课时学分/对象/简介/思政目标/评价/一致性说明）")
		}
	}
	if len(s.Objectives) == 0 || len(s.ContentSchedule) == 0 || len(s.Resources) == 0 {
		return fmt.Errorf("大纲课程目标、内容学时安排、教材资源不能为空")
	}
	for _, o := range s.Objectives {
		if strings.TrimSpace(o) == "" {
			return fmt.Errorf("大纲课程目标存在空项")
		}
	}
	for _, r := range s.Resources {
		if strings.TrimSpace(r) == "" {
			return fmt.Errorf("大纲教材资源存在空项")
		}
	}
	hasAIStage := false
	for _, u := range s.ContentSchedule {
		if strings.TrimSpace(u.Unit) == "" || strings.TrimSpace(u.Hours) == "" || strings.TrimSpace(u.KeyPoints) == "" {
			return fmt.Errorf("大纲内容学时安排须含单元/学时/重难点")
		}
		if strings.TrimSpace(u.AIStage) != "" {
			hasAIStage = true
		}
	}
	if ai {
		if strings.TrimSpace(s.AIIntegration) == "" {
			return fmt.Errorf("AI赛道大纲须含AI融合说明（平台/智能体如何进入目标、内容、评价）")
		}
		if !hasAIStage {
			return fmt.Errorf("AI赛道大纲内容学时安排须至少一个单元标注AI应用环节")
		}
	}
	body := strings.Join([]string{s.CourseName, s.CourseNature, s.CreditsHours, s.Audience, s.Overview, s.IdeologyPolitics, s.Prerequisites, s.Assessment, s.AIIntegration, s.Consistency}, "\n")
	body += "\n" + strings.Join(s.Objectives, "\n") + "\n" + strings.Join(s.Resources, "\n")
	for _, u := range s.ContentSchedule {
		body += "\n" + u.Unit + " " + u.Hours + " " + u.KeyPoints + " " + u.AIStage
	}
	if syllabusFactRisk.MatchString(body) {
		return fmt.Errorf("大纲含成效断言（提升率/满意度/获奖/采用等）；用户事实用待补占位，学时学分占比等结构数据除外")
	}
	if !anonymousPresentation(body) {
		return fmt.Errorf("竞赛版大纲须匿名：不得出现姓名/学校/院系（模板中的任课教师/开课单位字段不填身份）")
	}
	for _, id := range syllabusPartIDs {
		ids[id] = true
	}
	p.Warnings = nil
	if strings.Contains(s.CreditsHours, "待补") {
		p.Warnings = append(p.Warnings, "学分学时为待补占位：请按教务系统实际数据核实后填写。")
	}
	if ai && !strings.Contains(strings.Join(s.Objectives, " "), "数字素养") && !strings.Contains(strings.Join(s.Objectives, " "), "AI") && !strings.Contains(strings.Join(s.Objectives, " "), "智能") {
		p.Warnings = append(p.Warnings, "建议课程目标显式写入数字素养/AI能力/智能伦理（AI赛道视频评分要点），不阻断。")
	}
	return nil
}

const syllabusPrompt = `你是课程教学大纲架构师，只生成竞赛版大纲框架草案，不冒称已定稿。本轮kind=syllabus。sections/timeline/demo_script/lesson_annotations/slides为空数组，word_budget_total/duration_minutes/demo_duration_minutes为零。另加syllabus对象，字段：{"course_name":"课程名称","course_nature":"课程性质与专业地位","credits_hours":"学分/学时（理论/实践分配，未知用待补）","audience":"授课对象（未知人数用待补）","overview":"课程简介（不少于200字思路）","objectives":["可测量的课程目标，知识/能力/素养"],"ideology_politics":"课程思政目标","prerequisites":"先修/并进/后续课程（可选）","content_schedule":[{"unit":"章节单元","hours":"学时分配","key_points":"重难点","ai_stage":"本单元AI应用环节（AI赛道必填至少一处）"}],"assessment":"考核方式与成绩构成（形成性+终结性，占比结构）","resources":["教材（作者/出版社/版次）","参考文献","网络资源/平台/智能体"],"ai_integration":"AI融合说明（AI赛道必填）","consistency":"与报告/实录/教案课件的一致性说明"}。
依据：省赛通知配套材料明文要求大纲含课程名称/性质/课时学分/学生对象/简介/目标/内容与教学安排/评价/教材资源等核心要素（河南第七届九要素、海南八要素+等）；首届湖南通知要求大纲"反映参赛教师教学思想、课程设计思路和教学特色"；浙江音乐学院公开的竞赛大纲模板另列课程思政目标、先修后续课程、实践环节、成绩评定占比表，本框架采纳。竞赛版大纲须匿名：模板里的任课教师/开课单位等身份字段不填姓名学校院系。
大纲框架各赛道通用，内容按赛道侧重：人工智能赛道objectives须含数字素养/AI能力/智能伦理，content_schedule标注AI应用环节，assessment含多维智能评价/过程数据，resources含智慧教育平台或教学智能体，ai_integration必填；课程思政赛道细化思政目标与专业融合点；产教融合赛道实践学时占比不低于30%并在学时分配体现；实验教学赛道写明实验类型（综合设计型/研究探索型）与理论/实验学时。
学时/学分/考核占比是结构数据，可给拟议分配并标注待用户核实；学生人数/成绩/提升率/满意度/获奖等用户事实一律待补占位，绝不编造。scoring_map的section_ids用固定id：syllabus:basic、syllabus:objectives、syllabus:ideology、syllabus:content、syllabus:assessment、syllabus:resources、syllabus:ai，恰好覆盖所给评分维度。references至少引用本轮提供的规则来源。`
package main

import (
	"strings"
	"testing"
)

func syllabusFixture() BuilderPlan {
	p := BuilderPlan{Kind: "syllabus", Line: "以人机协同判断力培养为主线", Loop: "目标→内容→评价→证据", Boundary: "待实施建议",
		Sections: []BuilderSection{}, Timeline: []BuilderStage{}, Demo: []BuilderDemo{}, Annotations: []BuilderAnnotation{},
		Evidence:   []string{"待补：教务系统学分学时与开课截图"},
		References: []BuilderReference{{"builtin:rules7:scoring", "借鉴评分维度对照大纲要素", ""}},
		Syllabus: &BuilderSyllabus{
			CourseName: "数据结构", CourseNature: "专业核心课", CreditsHours: "待补：学分/学时（拟议理论32/实践16）", Audience: "计算机类专业大二学生（人数待补）",
			Overview:         "本课程面向真实工程问题组织数据结构与算法教学，拟议简介待用户核实补充。",
			Objectives:       []string{"掌握线性表/树/图的结构与算法（知识）", "能针对真实问题选择并验证数据结构（能力）", "形成数字素养与负责任使用AI的意识（素养）"},
			IdeologyPolitics: "结合国产基础软件案例培养科技报国意识与工程伦理",
			Prerequisites:    "先修程序设计基础，后续操作系统/数据库",
			ContentSchedule:  []SyllabusUnit{{"线性表", "拟议8学时", "顺序/链式存储与操作实现", "AI辅助生成测试用例，学生核验边界条件"}, {"树与图", "拟议16学时", "遍历与最短路径", "AI只追问不代答，学生说明选择依据"}},
			Assessment:       "形成性（课堂核验记录/单元测验）拟议占40% + 终结性项目答辩拟议占60%（占比待用户核实）",
			Resources:        []string{"拟议教材：《数据结构》（作者/出版社/版次待补）", "国家高等教育智慧教育平台课程资源", "自建课程教学智能体（页面截图待补）"},
			AIIntegration:    "智能体进入课前诊断/课中追问核验/课后个性化练习；教师终评，AI不代答",
			Consistency:      "大纲目标/内容/评价与报告创新举措、实录节段、教案课件四项标注一一对应",
		}}
	for _, dim := range aiReportDimensions() {
		p.Scores = append(p.Scores, BuilderScore{dim, []string{"syllabus:objectives"}, []string{"待补：对应教学过程证据"}})
	}
	return p
}

func TestSyllabusBuilderValidAI(t *testing.T) {
	available := map[string]map[string]any{"builtin:rules7:scoring": {"authority_status": rules7Notice}}
	p, e := validateBuilderPlan(fixtureJSON(syllabusFixture()), "syllabus-architect", available)
	if e != nil {
		t.Fatal(e)
	}
	if p.Kind != "syllabus" || p.Syllabus == nil {
		t.Fatal("syllabus plan lost")
	}
}

func TestSyllabusBuilderRequiredElements(t *testing.T) {
	available := map[string]map[string]any{"builtin:rules7:scoring": {"authority_status": rules7Notice}}
	for _, mutate := range []func(*BuilderPlan){
		func(p *BuilderPlan) { p.Syllabus.Overview = "" },
		func(p *BuilderPlan) { p.Syllabus.IdeologyPolitics = "" },
		func(p *BuilderPlan) { p.Syllabus.Objectives = nil },
		func(p *BuilderPlan) { p.Syllabus.ContentSchedule = nil },
		func(p *BuilderPlan) { p.Syllabus.Resources = nil },
		func(p *BuilderPlan) { p.Syllabus.Consistency = "" },
		func(p *BuilderPlan) { p.Syllabus.ContentSchedule[0].Hours = "" },
		func(p *BuilderPlan) { p.Syllabus.AIIntegration = "" },
		func(p *BuilderPlan) {
			p.Syllabus.ContentSchedule[0].AIStage = ""
			p.Syllabus.ContentSchedule[1].AIStage = ""
		},
		func(p *BuilderPlan) { p.Syllabus.Overview = "学生满意度提升32%" },
		func(p *BuilderPlan) { p.Syllabus.CourseNature = "某某大学计算机学院核心课" },
		func(p *BuilderPlan) { p.Kind = "report" },
		func(p *BuilderPlan) { p.Sections = []BuilderSection{{ID: "s1", Title: "混入", WordBudget: 100, Questions: []string{"q"}, Evidence: []string{"待补：x"}}} },
	} {
		p := syllabusFixture()
		mutate(&p)
		if _, e := validateBuilderPlan(fixtureJSON(p), "syllabus-architect", available); e == nil {
			t.Fatalf("invalid syllabus accepted: %+v", p.Syllabus)
		}
	}
}

func TestSyllabusBuilderStructuralNumbersAllowed(t *testing.T) {
	available := map[string]map[string]any{"builtin:rules7:scoring": {"authority_status": rules7Notice}}
	p := syllabusFixture()
	p.Syllabus.CreditsHours = "4学分/48学时（理论32/实践16）"
	p.Syllabus.Assessment = "平时成绩占40%，期末项目答辩占60%"
	if _, e := validateBuilderPlan(fixtureJSON(p), "syllabus-architect", available); e != nil {
		t.Fatal("structural hours/weights rejected:", e)
	}
}

func TestSyllabusBuilderNonAITrack(t *testing.T) {
	policy, e := builderPolicyFor("新文科", 7)
	if e != nil {
		t.Fatal(e)
	}
	available := map[string]map[string]any{"builtin:新文科:report": {"authority_status": "第六届国赛正式文件，往届参考"}}
	p := syllabusFixture()
	p.Syllabus.AIIntegration = ""
	p.Syllabus.ContentSchedule[0].AIStage = ""
	p.Syllabus.ContentSchedule[1].AIStage = ""
	p.Syllabus.Objectives = []string{"掌握法律史基本脉络（知识）", "能运用史料论证观点（能力）", "形成法治意识与家国情怀（素养）"}
	p.Scores = nil
	for _, dim := range policy.Dimensions {
		p.Scores = append(p.Scores, BuilderScore{dim, []string{"syllabus:content"}, []string{"待补：对应教学过程证据"}})
	}
	p.References = []BuilderReference{{"builtin:新文科:report", "借鉴本赛道评分对照", ""}}
	if _, e := validateBuilderPlan(fixtureJSON(p), "syllabus-architect", available, policy); e != nil {
		t.Fatal("non-AI syllabus without AI fields rejected:", e)
	}
}

func TestSyllabusCannotMixIntoOtherRoles(t *testing.T) {
	available := map[string]map[string]any{"builtin:rules7:scoring": {"authority_status": rules7Notice}}
	p := builderFixture("architect")
	p.Syllabus = syllabusFixture().Syllabus
	if _, e := validateBuilderPlan(fixtureJSON(p), "architect", available); e == nil || !strings.Contains(e.Error(), "大纲") {
		t.Fatal("syllabus mixed into architect accepted", e)
	}
}

func TestSyllabusPromptAndRegistration(t *testing.T) {
	if !isBuilder("syllabus-architect") {
		t.Fatal("syllabus-architect not registered as builder")
	}
	prompt := generationPrompt("syllabus-architect", 7, "人工智能")
	for _, s := range []string{"kind=syllabus", "课程思政目标", "ai_integration", "匿名", "待补", "syllabus:objectives"} {
		if !strings.Contains(prompt, s) {
			t.Errorf("syllabus prompt missing %q", s)
		}
	}
	found := false
	for _, s := range teachingSpecialists {
		if s.ID == "syllabus-architect" {
			found = true
		}
	}
	if !found {
		t.Fatal("syllabus-architect missing from teachingSpecialists")
	}
	if len(defaultSpecialists) != 6 {
		t.Fatal("workspace mapping changed")
	}
}
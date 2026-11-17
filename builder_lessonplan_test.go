package main

import (
	"strings"
	"testing"
)

func lessonPlanFixture() BuilderPlan {
	p := BuilderPlan{Kind: "lessonplan", Line: "以人机协同判断力培养为主线", Loop: "目标→过程→评价→证据", Boundary: "待实施建议",
		Sections: []BuilderSection{}, Timeline: []BuilderStage{}, Demo: []BuilderDemo{}, Annotations: []BuilderAnnotation{},
		Evidence:   []string{"待补：学情前测数据与智能体后台截图"},
		References: []BuilderReference{{"builtin:rules7:scoring", "借鉴评分维度对照教案要素", ""}},
		LessonPlan: &BuilderLessonPlan{
			CourseName: "数据结构", SegmentTopic: "二叉树遍历", SegmentHours: "1学时（拟议45分钟，待核实）", Audience: "计算机类专业大二学生（人数待补）",
			Objectives:      []string{"掌握先序/中序/后序遍历（知识）", "能针对真实问题选择遍历策略并验证（能力）", "形成数字素养与负责任使用AI的意识（素养）"},
			LearnerAnalysis: "已学线性表，递归理解薄弱（前测数据待补）",
			KeyPoints:       "三种遍历的递归与实现", Difficulties: "递归执行过程与栈帧可视化",
			Stages: []LessonPlanStage{
				{Stage: "导入与前测", Minutes: "拟议5分钟", Teacher: "发布前测并诊断", Student: "完成前测说明思路", AI: "智能体采集前测生成学情画像", Intent: "以学定教", AIApplication: "课前诊断", AIPurpose: "定位递归薄弱点", AIOperation: "智能体出题→采集→画像→教师审核", AIExpected: "拟形成班级薄弱点分布（待实测）"},
				{Stage: "探究与核验", Minutes: "拟议25分钟", Teacher: "追问引导不代答", Student: "小组探究并核验AI解释", AI: "AI只追问不代答", Intent: "逼学生说明依据", AIApplication: "课中核验", AIPurpose: "训练判断力", AIOperation: "学生提交解释→AI追问→学生修订→教师终评", AIExpected: "拟提升解释质量（待实测）"},
				{Stage: "总结与延伸", Minutes: "拟议15分钟", Teacher: "总结点评", Student: "完成迁移任务", AI: "", Intent: "迁移巩固"},
			},
			Assessment:       "过程性（核验记录/前测后测对比）+ 迁移任务量规（占比待用户核实）",
			Resources:        "课件、智慧教育平台资源、课程智能体（页面截图待补）",
			IdeologyPolitics: "结合国产基础软件案例培养工程伦理与科技报国意识",
			Consistency:      "节段目标/过程/评价与大纲、报告创新举措、实录节段一一对应",
		}}
	for _, dim := range aiReportDimensions() {
		p.Scores = append(p.Scores, BuilderScore{dim, []string{"lessonplan:process"}, []string{"待补：对应教学过程证据"}})
	}
	return p
}

func TestLessonPlanBuilderValidAI(t *testing.T) {
	available := map[string]map[string]any{"builtin:rules7:scoring": {"authority_status": rules7Notice}}
	p, e := validateBuilderPlan(fixtureJSON(lessonPlanFixture()), "lessonplan-architect", available)
	if e != nil {
		t.Fatal(e)
	}
	if p.Kind != "lessonplan" || p.LessonPlan == nil {
		t.Fatal("lessonplan plan lost")
	}
}

func TestLessonPlanBuilderRequiredAndAIAnnotations(t *testing.T) {
	available := map[string]map[string]any{"builtin:rules7:scoring": {"authority_status": rules7Notice}}
	for _, mutate := range []func(*BuilderPlan){
		func(p *BuilderPlan) { p.LessonPlan.LearnerAnalysis = "" },
		func(p *BuilderPlan) { p.LessonPlan.Difficulties = "" },
		func(p *BuilderPlan) { p.LessonPlan.Objectives = nil },
		func(p *BuilderPlan) { p.LessonPlan.Stages = nil },
		func(p *BuilderPlan) { p.LessonPlan.Consistency = "" },
		func(p *BuilderPlan) { p.LessonPlan.Stages[0].Intent = "" },
		func(p *BuilderPlan) { p.LessonPlan.Stages[0].AIPurpose = "" },
		func(p *BuilderPlan) { p.LessonPlan.Stages[1].AIExpected = "" },
		func(p *BuilderPlan) {
			p.LessonPlan.Stages[0].AI = ""
			p.LessonPlan.Stages[1].AI = ""
		},
		func(p *BuilderPlan) { p.LessonPlan.Assessment = "学生满意度提升32%" },
		func(p *BuilderPlan) { p.LessonPlan.Resources = "某某大学计算机学院平台" },
		func(p *BuilderPlan) { p.Kind = "lesson" },
	} {
		p := lessonPlanFixture()
		mutate(&p)
		if _, e := validateBuilderPlan(fixtureJSON(p), "lessonplan-architect", available); e == nil {
			t.Fatalf("invalid lessonplan accepted: %+v", p.LessonPlan)
		}
	}
}

func TestLessonPlanBuilderStructuralMinutesAllowed(t *testing.T) {
	available := map[string]map[string]any{"builtin:rules7:scoring": {"authority_status": rules7Notice}}
	p := lessonPlanFixture()
	p.LessonPlan.SegmentHours = "1学时45分钟"
	p.LessonPlan.Stages[0].Minutes = "5分钟"
	p.LessonPlan.Assessment = "过程性评价占40%，迁移任务占60%"
	if _, e := validateBuilderPlan(fixtureJSON(p), "lessonplan-architect", available); e != nil {
		t.Fatal("structural minutes/weights rejected:", e)
	}
}

func TestLessonPlanBuilderNonAITrack(t *testing.T) {
	policy, e := builderPolicyFor("新文科", 7)
	if e != nil {
		t.Fatal(e)
	}
	available := map[string]map[string]any{"builtin:新文科:report": {"authority_status": "第六届国赛正式文件，往届参考"}}
	p := lessonPlanFixture()
	for i := range p.LessonPlan.Stages {
		p.LessonPlan.Stages[i].AI = ""
		p.LessonPlan.Stages[i].AIApplication = ""
		p.LessonPlan.Stages[i].AIPurpose = ""
		p.LessonPlan.Stages[i].AIOperation = ""
		p.LessonPlan.Stages[i].AIExpected = ""
	}
	p.LessonPlan.Objectives = []string{"掌握法律史脉络（知识）", "运用史料论证（能力）", "法治意识（素养）"}
	p.Scores = nil
	for _, dim := range policy.Dimensions {
		p.Scores = append(p.Scores, BuilderScore{dim, []string{"lessonplan:process"}, []string{"待补：对应教学过程证据"}})
	}
	p.References = []BuilderReference{{"builtin:新文科:report", "借鉴本赛道评分对照", ""}}
	if _, e := validateBuilderPlan(fixtureJSON(p), "lessonplan-architect", available, policy); e != nil {
		t.Fatal("non-AI lessonplan without AI rejected:", e)
	}
}

func TestLessonPlanCannotMixIntoOtherRoles(t *testing.T) {
	available := map[string]map[string]any{"builtin:rules7:scoring": {"authority_status": rules7Notice}}
	p := builderFixture("architect")
	p.LessonPlan = lessonPlanFixture().LessonPlan
	if _, e := validateBuilderPlan(fixtureJSON(p), "architect", available); e == nil || !strings.Contains(e.Error(), "教案") {
		t.Fatal("lessonplan mixed into architect accepted", e)
	}
}

func TestLessonPlanPromptAndRegistration(t *testing.T) {
	if !isBuilder("lessonplan-architect") {
		t.Fatal("lessonplan-architect not registered as builder")
	}
	prompt := generationPrompt("lessonplan-architect", 7, "人工智能")
	for _, s := range []string{"kind=lessonplan", "应用环节", "设计目的", "操作流程", "预期效果", "匿名", "待补", "lessonplan:process"} {
		if !strings.Contains(prompt, s) {
			t.Errorf("lessonplan prompt missing %q", s)
		}
	}
	found := false
	for _, s := range teachingSpecialists {
		if s.ID == "lessonplan-architect" {
			found = true
		}
	}
	if !found {
		t.Fatal("lessonplan-architect missing from teachingSpecialists")
	}
	if len(defaultSpecialists) != 7 {
		t.Fatal("workspace mapping changed")
	}
}
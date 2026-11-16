package main

import (
	"fmt"
	"net/http"
	"strings"
)

type TeachingSpecialist struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Purpose     string   `json:"purpose"`
	Checks      []string `json:"checks"`
	Deliverable string   `json:"deliverable"`
	Keywords    string   `json:"keywords"`
}

var teachingSpecialists = []TeachingSpecialist{
	{"chief", "备赛总顾问", "确定课程定位与材料准备顺序", []string{"真实教学问题是否有学情证据", "问题、举措、实施、评价、成效是否闭合", "先解决最影响论证的三项问题"}, "优先级清单、材料缺口与下一步", "教学问题 创新举措 成效"},
	{"report", "报告论证顾问", "检查报告论证和段落位置", []string{"课程概述是否挤占创新论证", "每项举措是否对应痛点并有实施细节", "章节重复、移位与先后顺序是否合理"}, "原文位置、问题、修改理由、可替换文本", "创新报告 真实问题 内容重构"},
	{"lesson", "课堂实录顾问", "检查课堂中创新是否真正发生", []string{"教师提问、学生思考与反馈形成怎样的互动链", "视频选段与报告主张是否一致", "区分可观察行为与仅有字幕无法证明的行为"}, "按时间列行为证据、缺口、可操作调整", "课堂实录 学生参与 教学实施"},
	{"presentation", "汇报与答辩顾问", "优化PPT叙事与现场表达", []string{"本页只有什么核心主张、证据和结论", "前后页是否重复或跳步，讲述时间是否合理", "评委最可能质疑什么，如何用已有证据回答"}, "页码、建议标题、讲稿要点、追问与答法；无画面不评价版式", "汇报 PPT 答辩 叙事"},
	{"evidence", "评价与成效顾问", "核查数据口径和评价证据", []string{"评价任务与学习目标是否一致", "样本、分母、时间、基线、量规是否明确", "区分使用次数、学习产物质量与因果结论"}, "证据链、数字口径风险、可补材料；不得编造统计", "学习成效 量规 评价 数据"},
	{"ai", "AI融合顾问", "检查人工智能改变教学的机制", []string{"为何需要AI而非一般数字工具", "学生、教师与AI各做什么判断和核验", "有无防代做、隐私伦理和能力提升的证据"}, "AI介入环节、人机分工、核验机制及无AI替代方案", "人工智能 人机协同 AI 教学"},
	{"values", "课程思政顾问", "将价值引领落实到专业活动", []string{"价值判断是否嵌在学科任务或专业决策", "学生是否需要举证、权衡并承担后果", "是否有学生产物而非只列思政口号"}, "专业任务、价值冲突、学生行动与评价对应表", "课程思政 价值引领 专业育人"},
	{"rules", "规则与材料核查顾问", "核对届次、赛道与材料要求", []string{"来源属于当届国赛、校赛还是往届参考", "报告、课堂实录与现场汇报要求是否混用", "主张是否能回溯到实际提供的原文件"}, "适用范围、来源和待补规则；缺当届文件不虚构硬性门槛", "评分标准 赛道 材料要求"},
}
var defaultSpecialists = map[string]string{"report": "report", "lesson": "lesson", "presentation": "presentation", "consistency": "evidence", "library": "rules", "replay": "chief", "overview": "chief"}
var trackFocus = map[string]string{
	"新工科":  "工程复杂问题、设计迭代、系统约束与真实工程产物。",
	"新医科":  "临床推理、患者安全、伦理与模拟/实践证据，不能把教学建议写成诊疗指令。",
	"新农科":  "农业真实情境、田间或产业任务、生态约束和实践证据。",
	"新文科":  "人文社会问题、文本/资料证据、价值判断与跨学科解释，不能照搬工程或AI案例。",
	"基础课程": "基础概念、思维方法、迁移能力与大班/差异化学习难点。",
	"课程思政": "专业知识与价值判断的内在关联及学生行为证据。",
	"产教融合": "产业需求、校企共同任务、协同育人与真实产出；合作协议不是全部成效。",
	"人工智能": "AI必要性、人机分工、学生判断力、责任伦理与防代做证据。",
	"实验教学": "实验设计、操作与探究、误差和安全、可复现实验记录。",
}

func (a *App) specialists(w http.ResponseWriter, r *http.Request) {
	jsonOut(w, 200, map[string]any{"roles": teachingSpecialists, "defaults": defaultSpecialists, "note": "预制AI专业分工，不是真人专家或官方评分规则。可叠加真实讲座专家的有据视角。"})
}
func specialistPrompt(id, workspace, track string) (string, error) {
	if id == "" || id == "auto" {
		id = defaultSpecialists[workspace]
		if id == "" {
			id = "chief"
		}
	}
	for _, s := range teachingSpecialists {
		if s.ID == id {
			text := "\n本轮专业分工：" + s.Name + "（预制AI角色，不是真人身份）。任务：" + s.Purpose + "。检查顺序：" + strings.Join(s.Checks, "；") + "。交付：" + s.Deliverable + "。只执行用户本轮需求，不因为角色而扩大到全部材料。若要求JSON，仍严格使用既定JSON结构，将专业检查写入comment，不增加字段。公开专家的观点只能来自实际检索证据；用户事实、讲座证据和你的推断分开。\n"
			if focus := trackFocus[track]; focus != "" {
				text += "当前赛道的教学研究侧重点（AI建议，不是官方权重）：" + track + "：" + focus
			}
			return text, nil
		}
	}
	return "", fmt.Errorf("未知备赛专业分工，请重新选择")
}

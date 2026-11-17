package main

import (
	"context"
	"fmt"
	"strings"
)

const deepFile = "knowledge/ai_track_deep.json"
const deepNotice = "专家方法＝36场迁移建议非专家本人观点、非官方标准；第六届案例＝公开报道转述非报告全文、非用户成效。"

func installDeepReferences(add func(builtinReference) error) error {
	var deep map[string]any
	if err := readReferenceJSON("knowledge/ai_track_deep.json", &deep); err != nil {
		return err
	}
	for _, group := range []string{"priority_sessions", "support_sessions", "sixth_edition_reports"} {
		for _, x := range objects(deep[group]) {
			speaker := stringField(x, "speaker")
			if speaker == "" {
				speaker = stringField(x, "teacher")
			}
			r := builtinReference{ID: "builtin:aideep:" + stringField(x, "id"), Kind: "ai_migration", Title: speaker + " · " + stringField(x, "course"), Text: referenceText(x), Authority: deepNotice, SourceFile: deepFile, SourceURL: repoSource(deepFile), Publisher: "已审计的讲座研究与公开报道", Origin: speaker, SourceLocator: stringField(x, "source_locator"), SourceNature: stringField(x, "source_nature"), Verification: stringField(x, "verification"), SourceGroup: group, ReportURL: stringField(x, "source_url"), PracticeURL: stringField(x, "practice_source_url")}
			if group != "sixth_edition_reports" {
				brief := speaker + " 可选教学机制（迁移建议）：\n"
				for _, move := range objects(x["moves"]) {
					brief += stringField(move, "name") + "：" + stringField(move, "original") + "；边界：" + stringField(move, "caution") + "；定位：" + stringField(move, "source_locator") + "\n"
				}
				r.Text = brief + "\n完整记录：" + r.Text
			}
			if group == "sixth_edition_reports" {
				r.Edition = 6
				r.Title += " · " + stringField(x, "award")
				r.Authority += " 往届公开报道，不是你的课程事实。来源性质：" + r.SourceNature
				if r.Verification == "transcribed" {
					r.Authority += " 据公开报道/复盘转述，不得写成获奖者逐字原话。"
				}
			}
			if speaker == "刘学莘" {
				r.Authority += " 个人评审经验，非官方评分条文。"
			}
			if speaker == "李静" && group == "priority_sessions" {
				r.Authority += " 创新设计占时过高，成效证据偏少；引用机制须补齐成效数据。"
			}
			if stringField(x, "id") == "gao_yue_ai_fit" {
				r.Authority += " 三维痛点定位法/AI适配度象限仅有名称，不得编造具体维度或定义。"
			}
			if err := add(r); err != nil {
				return err
			}
		}
	}
	return nil
}

// Priority records precede supplementary evidence so a small context still has
// a real teaching mechanism and its citation boundary, rather than only rules.
func builderDeepEvidence(ctx context.Context, q string) []map[string]any {
	out := []map[string]any{}
	for _, id := range []string{"lijing_ai_track", "songxiaofei_ai_track"} {
		if r := builtinReferenceByID(ctx, "builtin:aideep:"+id); r != nil {
			out = append(out, r)
		}
	}
	for _, r := range builtinReferenceSearch(ctx, q) {
		if strings.HasPrefix(stringField(r, "id"), "builtin:aideep:") && stringField(r, "source_group") != "priority_sessions" {
			out = append(out, r)
			if len(out) >= 4 {
				break
			}
		}
	}
	return out
}
func builtinReferenceByID(ctx context.Context, id string) map[string]any {
	db, e := builtinLibrary()
	if e != nil {
		return nil
	}
	var raw string
	if db.QueryRowContext(ctx, "SELECT record FROM builtin_reference WHERE id=?", id).Scan(&raw) != nil {
		return nil
	}
	var r builtinReference
	if decodeReference(raw, &r) != nil {
		return nil
	}
	return referenceMap(r)
}
func validateDeepCitations(p BuilderPlan, available map[string]map[string]any, ai bool) error {
	hasPriority, usedPriority := false, false
	for _, r := range available {
		if stringField(r, "source_group") == "priority_sessions" {
			hasPriority = true
		}
	}
	text := p.Line + "\n" + p.Loop
	for _, s := range p.Sections {
		text += "\n" + s.Title + "\n" + s.Draft + "\n" + strings.Join(s.Questions, "\n")
	}
	for _, r := range p.References {
		text += "\n" + r.Use
	}
	seen := map[string]bool{}
	for _, r := range p.References {
		if seen[r.ID] {
			return fmt.Errorf("同一来源重复引用，请合并对应打法")
		}
		seen[r.ID] = true
		src := available[r.ID]
		if strings.HasPrefix(r.ID, "builtin:aideep:") && !ai {
			return fmt.Errorf("非AI赛道不得注入AI深度资料")
		}
		if stringField(src, "source_group") == "priority_sessions" {
			usedPriority = true
		}
		if stringField(src, "verification") == "transcribed" && hasAny(text, "原话", "原句", "逐字", "说：", "说:") {
			return fmt.Errorf("转述来源不得写成逐字原话")
		}
		if strings.HasSuffix(r.ID, ":gao_yue_ai_fit") && !strings.Contains(r.Use, "仅名称") && !strings.Contains(r.Use, "细节见其本人分享") {
			return fmt.Errorf("高越方法仅名称可核，不得扩写维度；请明确细节见其本人分享")
		}
	}
	if ai && hasPriority && !usedPriority {
		return fmt.Errorf("AI生成需引用本轮李静或宋晓菲的深度资料机制")
	}
	return nil
}

// Exclude overlap at retrieval as well as in the prompt; distinct mechanisms
// (e.g. Zhang Jing refusal and Xin Ye blind review) remain available.
func duplicateDeepPlay(id string, selected []map[string]any) bool {
	overlap := map[string]string{"builtin:play:six_hats_black_hat": "songxiaofei_ai_track", "builtin:play:low_cost_tool_view": "songxiaofei_ai_track", "builtin:play:three_checks_benchmark": "yuxinjie_ai_logic", "builtin:play:ai_tiered_governance": "yuxinjie_ai_logic", "builtin:play:judge-insights": "liuxueshen_ai_judges"}
	target := overlap[id]
	if target == "" {
		return false
	}
	for _, r := range selected {
		if stringField(r, "id") == "builtin:aideep:"+target {
			return true
		}
	}
	return false
}

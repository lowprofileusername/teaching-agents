package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// Public, versioned reference material only. No user reports or lecture media.
//
//go:embed knowledge/rules.json knowledge/rules_6_2026.pdf knowledge/experts.json knowledge/rules_7_ai_2026.json knowledge/ai_playbook.json knowledge/ai_award_cases_6th.json knowledge/video_index_5th_wenke.json research/*.md research/third-party/*.md
var builtinFiles embed.FS

type RulePart struct {
	Score      int      `json:"score"`
	Page       int      `json:"page"`
	Dimensions []string `json:"dimensions"`
}
type RuleGroup struct {
	ID                string              `json:"id"`
	Tracks            []string            `json:"tracks"`
	Focus             string              `json:"focus"`
	VideoRequirements string              `json:"video_requirements"`
	Parts             map[string]RulePart `json:"parts"`
}
type RuleCatalog struct {
	Source       map[string]any `json:"source"`
	FutureNote   string         `json:"future_note"`
	FutureSource string         `json:"future_source"`
	Groups       []RuleGroup    `json:"groups"`
}

var builtinState struct {
	sync.Once
	db      *sql.DB
	err     error
	catalog RuleCatalog
}

func builtinLibrary() (*sql.DB, error) {
	builtinState.Do(func() {
		b, e := builtinFiles.ReadFile("knowledge/rules.json")
		if e == nil {
			e = json.Unmarshal(b, &builtinState.catalog)
		}
		if e != nil {
			builtinState.err = e
			return
		}
		db, e := sql.Open(sqliteDriver, ":memory:")
		if e != nil {
			builtinState.err = e
			return
		}
		db.SetMaxOpenConns(1)
		_, e = db.Exec(`PRAGMA temp_store=MEMORY; CREATE TABLE rule(id TEXT PRIMARY KEY,track TEXT,part TEXT,title TEXT,pdf_page_no INTEGER,text TEXT);`)
		if e == nil {
			for _, g := range builtinState.catalog.Groups {
				for _, track := range g.Tracks {
					for _, part := range []string{"report", "lesson", "presentation"} {
						p := g.Parts[part]
						label := map[string]string{"report": "创新成果报告", "lesson": "课堂实录", "presentation": "教学设计创新汇报"}[part]
						body := fmt.Sprintf("第六届国赛参考：%s赛道 / %s占%d分。评分维度：%s。文件未提供这些维度的细分权重，不编造分项分值。\n%s\n实录形式要求：%s\n%s", track, label, p.Score, strings.Join(p.Dimensions, "、"), g.Focus, g.VideoRequirements, builtinState.catalog.FutureNote)
						_, e = db.Exec("INSERT INTO rule VALUES(?,?,?,?,?,?)", "builtin:"+track+":"+part, track, part, track+"赛道 · "+label, p.Page, body)
						if e != nil {
							break
						}
					}
					if e != nil {
						break
					}
				}
				if e != nil {
					break
				}
			}
		}
		if e == nil {
			e = installExperts(db)
		}
		if e == nil {
			e = installBuiltinReferences(db)
		}
		if e == nil {
			_, e = db.Exec("PRAGMA query_only=ON")
		}
		if e != nil {
			db.Close()
			builtinState.err = e
			return
		}
		builtinState.db = db
	})
	return builtinState.db, builtinState.err
}
func (a *App) builtinRules(w http.ResponseWriter, r *http.Request) {
	if _, e := builtinLibrary(); e != nil {
		fail(w, 500, "内置规则加载失败")
		return
	}
	jsonOut(w, 200, builtinState.catalog)
}
func builtinRuleSearch(ctx context.Context, q string) []map[string]any {
	if queryEdition(q) == 7 && queryAI(q) {
		return nil
	}
	db, e := builtinLibrary()
	if e != nil {
		return []map[string]any{}
	}
	track := ""
	for _, g := range builtinState.catalog.Groups {
		for _, t := range g.Tracks {
			if strings.Contains(q, t) {
				track = t
			}
		}
	}
	if strings.Contains(strings.ToLower(q), "ai赛道") {
		track = "人工智能"
	}
	if strings.Contains(q, "未选赛道") {
		track = ""
	}
	part := ""
	if strings.Contains(q, "课堂") || strings.Contains(q, "实录") {
		part = "lesson"
	} else if strings.Contains(strings.ToLower(q), "ppt") || strings.Contains(q, "汇报") {
		part = "presentation"
	} else if strings.Contains(q, "报告") {
		part = "report"
	}
	relevant := track != "" || part != ""
	for _, s := range []string{"赛道", "评分", "标准", "比赛", "教创赛", "备赛", "规则", "答辩"} {
		if strings.Contains(q, s) {
			relevant = true
		}
	}
	if !relevant {
		return []map[string]any{}
	}
	// With no track, return group representatives, not nine near-duplicate records.
	rows, e := rowsJSON(ctx, db, `SELECT * FROM rule WHERE (?='' OR track=?) AND (?='' OR part=?) ORDER BY CASE track WHEN '新工科' THEN 0 WHEN '课程思政' THEN 1 WHEN '产教融合' THEN 2 WHEN '人工智能' THEN 3 WHEN '实验教学' THEN 4 ELSE 5 END,part`, track, track, part, part)
	if e != nil {
		return []map[string]any{}
	}
	out := []map[string]any{}
	for _, row := range rows {
		if track == "" && oneOf(fmt.Sprint(row["track"]), "新医科", "新农科", "新文科", "基础课程") {
			continue
		}
		row["document_id"] = "builtin:rules6"
		row["title"] = "第六届国赛参考 · " + fmt.Sprint(row["title"])
		row["kind"] = "competition_rule"
		row["edition"] = 6
		row["authority_status"] = builtinState.catalog.Source["status"]
		row["source_url"] = builtinState.catalog.Source["url"]
		row["publisher"] = builtinState.catalog.Source["publisher"]
		row["extraction"] = "原PDF图像核对后的摘要，非逐字引文"
		row["pdf_available"] = true
		out = append(out, row)
		if len(out) >= 6 {
			break
		}
	}
	return out
}

func builtinSearch(ctx context.Context, q string) []map[string]any {
	out := append(builtinReferenceSearch(ctx, q), builtinRuleSearch(ctx, q)...)
	return append(out, builtinExpertSearch(ctx, q)...)
}

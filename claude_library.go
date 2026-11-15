package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// The source file is opened mode=ro. Only small text tables and indexes are
// normalized in SQLite TEMP memory; media and the original schema stay intact.
func prepareClaudeLibrary(ctx context.Context, db *sql.DB) (bool, error) {
	cols, err := rowsJSON(ctx, db, `PRAGMA main.table_info(video)`)
	if err != nil {
		return false, err
	}
	claude := false
	for _, c := range cols {
		if c["name"] == "speaker" {
			claude = true
		}
	}
	if !claude {
		return false, nil
	}
	if _, err = db.ExecContext(ctx, `PRAGMA query_only=OFF`); err != nil {
		return false, err
	}
	defer db.Exec(`PRAGMA query_only=ON`)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	statements := []string{
		`CREATE TEMP TABLE video AS SELECT tag,speaker AS teacher,school AS institution,course,talk_title AS title,duration_s AS duration,asr_engine,asr_quality,method,provenance,session,track FROM main.video`,
		`CREATE UNIQUE INDEX temp.claude_video_tag ON video(tag)`,
		`CREATE TEMP TABLE page AS SELECT tag||':P'||page_no AS id,id AS source_row_id,tag,page_no,NULL AS original_page_no,title,coalesce(ocr_text,'') AS page_text,section,t_start,t_end,time_ranges,main_message,elements,function,relation,visual,evidence,borrow,migrate,speech_text,speech_sentences,length(image)>0 AS has_image FROM main.page`,
		`CREATE UNIQUE INDEX temp.claude_page_id ON page(id)`,
		`CREATE INDEX temp.claude_page_number ON page(tag,page_no)`,
		`CREATE TEMP TABLE utterance AS SELECT cast(u.id AS TEXT) AS id,u.tag,u.page_no,u.t_start AS t0,u.t_end AS t1,u.text,coalesce(v.asr_engine,'ASR') AS version,'来源库标注：'||coalesce(v.asr_quality,'未整场听校') AS quality_status,v.speaker AS teacher_attribution FROM main.utterance u JOIN main.video v ON v.tag=u.tag`,
		`CREATE INDEX temp.claude_utter_page ON utterance(tag,page_no,t0)`,
		`CREATE TEMP VIEW page_utterance AS SELECT p.id AS page_id,u.id AS utterance_id FROM temp.page p JOIN temp.utterance u ON u.tag=p.tag AND u.page_no=p.page_no`,
		`CREATE TEMP TABLE page_occurrence AS SELECT p.id AS page_id,cast(json_extract(j.value,'$[0]') AS REAL) AS t0,cast(json_extract(j.value,'$[1]') AS REAL) AS t1 FROM temp.page p,json_each(CASE WHEN json_valid(p.time_ranges) AND json_array_length(p.time_ranges)>0 THEN p.time_ranges ELSE json_array(json_array(p.t_start,p.t_end)) END) j`,
		`CREATE INDEX temp.claude_occurrence_page ON page_occurrence(page_id,t0)`,
		`CREATE TEMP TABLE research_card AS SELECT 'claude:insight:'||id AS id,kind,title,json_object('body',body,'sources',sources,'evidence_tier','来源库研究判断，非讲者原话') AS payload_json FROM main.insight`,
		`CREATE TEMP TABLE claude_adapter(format TEXT)`,
		`INSERT INTO claude_adapter VALUES('claude')`,
	}
	for _, s := range statements {
		if _, err = tx.ExecContext(ctx, s); err != nil {
			return false, fmt.Errorf("Claude资料库适配失败：%w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func isClaudeLibrary(db *sql.DB) bool {
	var n int
	return db.QueryRow(`SELECT 1 FROM temp.claude_adapter LIMIT 1`).Scan(&n) == nil
}

func claudePage(ctx context.Context, db *sql.DB, id string) (map[string]any, error) {
	rows, err := rowsJSON(ctx, db, `SELECT p.*,v.teacher,v.institution,v.course FROM temp.page p JOIN temp.video v ON v.tag=p.tag WHERE p.id=?`, id)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("页面不存在")
	}
	p := rows[0]
	p["reference"] = id
	p["source_format"] = "claude"
	p["page_number_note"] = "P为此库去重页顺序，不是原PPT角标；不可与另一库的R编号混用"
	for _, key := range []string{"occurrences", "utterances", "issues", "images", "official_pages", "annotations"} {
		p[key] = []map[string]any{}
	}
	for _, s := range []struct {
		key, q string
		args   []any
	}{
		{"occurrences", `SELECT t0,t1 FROM temp.page_occurrence WHERE page_id=? ORDER BY t0`, []any{id}},
		{"utterances", `SELECT * FROM temp.utterance WHERE tag=? AND page_no=? ORDER BY t0`, []any{p["tag"], p["page_no"]}},
		{"annotations", `SELECT wrong AS original,right AS candidate,found_by AS basis,'来源库勘误（场次级，非本程序独立核验）' AS status FROM main.errata WHERE tag=?`, []any{p["tag"]}},
	} {
		p[s.key], err = rowsJSON(ctx, db, s.q, s.args...)
		if err != nil {
			return nil, err
		}
	}
	if fmt.Sprint(p["has_image"]) == "1" {
		p["images"] = []map[string]any{{"asset_id": fmt.Sprint("claude:page-image:", p["source_row_id"]), "source_seconds": p["t_start"], "kind": "来源库代表截图"}}
	}
	analysis := map[string]any{}
	for _, k := range []string{"section", "main_message", "elements", "function", "relation", "visual", "evidence", "borrow", "migrate"} {
		analysis[k] = p[k]
	}
	p["analysis"] = analysis
	p["research_analysis_unverified"] = fmt.Sprintf("来源库研究标注（非画面逐字内容、非讲者原话）：主信息=%v；教学功能=%v；可借鉴=%v；迁移建议=%v", p["main_message"], p["function"], p["borrow"], p["migrate"])
	return p, nil
}

// Main-table rowids keep media lookups indexed; never scan or cache BLOBs during search.
func claudeAsset(ctx context.Context, db *sql.DB, id string) (string, []byte, error) {
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 || parts[0] != "claude" {
		return "", nil, errors.New("媒体编号无效")
	}
	var q, mime string
	var arg any = parts[2]
	switch parts[1] {
	case "page-image":
		q = `SELECT image FROM main.page WHERE id=?`
		mime = "image/jpeg"
	case "frame-image":
		q = `SELECT image FROM main.frame WHERE id=?`
		mime = "image/jpeg"
	case "flash-image":
		q = `SELECT image FROM main.flash WHERE id=?`
		mime = "image/jpeg"
	case "video-audio":
		q = `SELECT audio FROM main.video WHERE tag=?`
		mime = "audio/ogg"
	default:
		return "", nil, errors.New("不支持的媒体类型")
	}
	if parts[1] != "video-audio" {
		n, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return "", nil, err
		}
		arg = n
	}
	var b []byte
	err := db.QueryRowContext(ctx, q, arg).Scan(&b)
	if err == nil && len(b) == 0 {
		err = errors.New("媒体为空")
	}
	return mime, b, err
}

func claudeAudio(w http.ResponseWriter, r *http.Request, db *sql.DB, tag string, t float64) {
	var duration float64
	var size int
	err := db.QueryRowContext(r.Context(), `SELECT duration_s,length(audio) FROM main.video WHERE tag=?`, tag).Scan(&duration, &size)
	if err != nil || size == 0 || t < 0 || t >= duration {
		fail(w, 404, "该时刻没有库内原声")
		return
	}
	// Use full-session audio to preserve video time even on a page with disjoint
	// revisits: concatenated per-page audio has a different time axis.
	jsonOut(w, 200, map[string]any{"asset_id": "claude:video-audio:" + tag, "t0": 0, "t1": duration, "source_audio_start": 0, "encoded_duration": duration, "quality_status": "来源库整场音频；按视频时间定位，未整场听校"})
}

func claudeRawOCR(ctx context.Context, db *sql.DB, q, tag string) []map[string]any {
	terms := searchTerms(q)
	out := []map[string]any{}
	if len(terms) == 0 {
		return out
	}
	if len(terms) > 4 {
		terms = terms[:4]
	}
	parts := []string{}
	args := []any{tag, tag}
	for _, t := range terms {
		parts = append(parts, "instr(text,?)>0")
		args = append(args, t)
	}
	query := `SELECT tag,seconds,text_id,text FROM (
 SELECT tag,mid AS seconds,'flash:'||id AS text_id,text FROM main.flash
 UNION ALL SELECT tag,t,'second_full:'||tag||':'||t,ocr_text FROM main.second_full
 UNION ALL SELECT s.tag,s.t,'second:'||s.tag||':'||s.t,s.ocr_text FROM main.second s WHERE NOT EXISTS(SELECT 1 FROM main.second_full f WHERE f.tag=s.tag AND f.t=s.t)
 ) WHERE (?='' OR tag=?) AND (` + strings.Join(parts, " OR ") + `) LIMIT 12`
	rows, err := rowsJSON(ctx, db, query, args...)
	if err == nil {
		return rows
	}
	return out
}

// Resolve existing canonical search hints against this database's actual tags.
func libraryTagHints(vs []map[string]any, named map[string]bool) map[string]bool {
	aliases := map[string]string{"songxiaofei": "song", "lijing": "li", "liuxueshen_xgk": "liuxs", "caominhui": "caomh", "yumeifang": "yumf", "zhangjing": "zhangj", "xiangmiaolian": "xiangml", "liuzhanhe": "liuzh"}
	available := map[string]bool{}
	for _, v := range vs {
		available[fmt.Sprint(v["tag"])] = true
	}
	for tag := range named {
		if !available[tag] && available[aliases[tag]] {
			delete(named, tag)
			named[aliases[tag]] = true
		}
	}
	return named
}

func claudeReadme(ctx context.Context, db *sql.DB) string {
	rows, err := rowsJSON(ctx, db, `SELECT key,body FROM main.readme`)
	if err != nil {
		return ""
	}
	b, _ := json.Marshal(rows)
	return string(b)
}

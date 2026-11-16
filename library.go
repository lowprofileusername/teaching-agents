package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var libraryState struct {
	sync.RWMutex
	db   *sql.DB
	path string
}

const projectFolder = "第五届高校智慧教育与教学创新成果交流会"

func rowsJSON(ctx context.Context, db *sql.DB, q string, args ...any) ([]map[string]any, error) {
	rows, e := db.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	cols, e := rows.Columns()
	if e != nil {
		return nil, e
	}
	out := []map[string]any{}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptr := make([]any, len(cols))
		for i := range vals {
			ptr[i] = &vals[i]
		}
		if e = rows.Scan(ptr...); e != nil {
			return nil, e
		}
		r := map[string]any{}
		for i, k := range cols {
			if b, ok := vals[i].([]byte); ok {
				r[k] = string(b)
			} else {
				r[k] = vals[i]
			}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
func openLibrary(path string) error {
	abs, e := filepath.Abs(strings.TrimSpace(path))
	if e != nil {
		return e
	}
	info, e := os.Stat(abs)
	if e != nil || info.IsDir() {
		return errors.New("找不到数据库文件，请填写已挂载共享盘上的 SQLite 文件路径")
	}
	upath := filepath.ToSlash(abs)
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
	}
	u := url.URL{Scheme: "file", Path: upath}
	q := url.Values{}
	q.Set("mode", "ro")
	q.Set("_query_only", "true")
	q.Set("_busy_timeout", "3000")
	for _, p := range []string{"query_only(1)", "temp_store(2)", "cache_size(-16384)", "busy_timeout(3000)", "trusted_schema(0)"} {
		q.Add("_pragma", p)
	}
	u.RawQuery = q.Encode()
	db, e := sql.Open(sqliteDriver, u.String())
	if e != nil {
		return e
	}
	db.SetMaxOpenConns(1)
	if _, e = db.Exec("PRAGMA query_only=ON; PRAGMA temp_store=MEMORY; PRAGMA cache_size=-16384; PRAGMA trusted_schema=OFF;"); e != nil {
		db.Close()
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	claude, e := prepareClaudeLibrary(ctx, db)
	if e != nil {
		db.Close()
		return e
	}
	required := []string{"video", "page", "page_occurrence", "utterance", "asset", "page_image", "audio_chunk"}
	if claude {
		required = []string{"video", "page", "page_occurrence", "utterance"}
	}
	for _, t := range required {
		var n int
		if e = db.QueryRowContext(ctx, "SELECT 1 FROM "+t+" LIMIT 1").Scan(&n); e != nil && e != sql.ErrNoRows {
			db.Close()
			return fmt.Errorf("数据库缺少可读取的 %s 表：%w", t, e)
		}
	}
	libraryState.Lock()
	old := libraryState.db
	libraryState.db = db
	libraryState.path = abs
	libraryState.Unlock()
	if old != nil {
		old.Close()
	}
	return nil
}
func autoLibrary() {
	rel := filepath.Join("codex_analysis_tmp", "qa", "compact_database", "expert_videos_v1.sqlite3")
	exe, _ := os.Executable()
	paths := []string{filepath.Join(filepath.Dir(exe), "..", "..", "..", rel), filepath.Join("/Volumes/Share", projectFolder, rel), filepath.Join(`Z:\`, projectFolder, rel), filepath.Join(`\\10.0.0.3\Share`, projectFolder, rel)}
	if custom := os.Getenv("TEACHING_LIBRARY_DB"); custom != "" {
		paths = append([]string{custom}, paths...)
	}
	for _, p := range paths {
		if _, e := os.Stat(p); e == nil {
			if openLibrary(p) == nil {
				return
			}
		}
	}
}
func getLibrary() (*sql.DB, string, error) {
	libraryState.RLock()
	defer libraryState.RUnlock()
	if libraryState.db == nil {
		return nil, "", errors.New("尚未连接资料库，请在资料库设置中填写 SQLite 路径")
	}
	return libraryState.db, libraryState.path, nil
}
func (a *App) library(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Path string `json:"path"`
	}
	if !a.read(w, r, &x) {
		return
	}
	if x.Path != "" {
		if e := openLibrary(x.Path); e != nil {
			fail(w, 400, e.Error())
			return
		}
	}
	db, path, e := getLibrary()
	if e != nil {
		jsonOut(w, 200, map[string]any{"path": "", "format": "builtin", "bytes": 0, "videos": []any{}, "counts": map[string]int{"video": 0, "page": 0, "utterance": 0, "builtin_rules": 27}, "coverage_note": "内置第六届9赛道参考可用；未连接外部讲座库"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	videos, e := rowsJSON(ctx, db, `SELECT v.tag,v.teacher,v.institution,v.course,v.title,v.duration,count(p.id) as page_count FROM video v LEFT JOIN page p ON p.tag=v.tag GROUP BY v.tag ORDER BY v.teacher`)
	if e != nil {
		fail(w, 500, e.Error())
		return
	}
	counts := map[string]int{}
	for _, t := range []string{"video", "page", "utterance", "audio_chunk", "source_document", "source_document_page", "knowledge_document", "knowledge_document_page", "insight", "frame", "flash", "second", "second_full", "plan_a_done", "errata"} {
		var n int
		if e = db.QueryRowContext(ctx, "SELECT count(*) FROM "+t).Scan(&n); e == nil {
			counts[t] = n
		}
	}
	info, _ := os.Stat(path)
	pathSaved := true
	if x.Path != "" {
		pathSaved = a.saveLibraryPath(path)
	}
	format, note, readme := "codex", "R是检索单元编号，不一定是原PPT页码", ""
	if isClaudeLibrary(db) {
		format = "claude"
		note = "P是此库去重页顺序，不是原PPT角标；研究分析与转写不等于原声听校"
		readme = claudeReadme(ctx, db)
	}
	jsonOut(w, 200, map[string]any{"format": format, "coverage_note": note, "source_readme": readme, "path_saved": pathSaved, "path": path, "bytes": info.Size(), "read_only": true, "videos": videos, "counts": counts})
}

var hanWords = regexp.MustCompile(`[\p{Han}]+|[a-zA-Z0-9_]+`)

func searchTerms(q string) []string {
	for _, s := range []string{"有什么", "第几页", "第十二页", "具体", "怎么样", "如何", "哪些", "什么", "为什么", "请问", "老师", "讲了", "说了", "内容", "介绍", "一下", "总结", "分享", "之间", "区别", "共同", "对比", "不要联网", "联网检索", "只用数据库", "数据库里的", "根据讲座", "回答我", "请说明", "请分别", "请给出", "结合我上传的", "结合本地研究报告"} {
		q = strings.ReplaceAll(q, s, " ")
	}
	out := []string{}
	seen := map[string]bool{}
	add := func(s string) {
		if len([]rune(s)) >= 2 && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	for _, s := range hanWords.FindAllString(strings.ToLower(q), -1) {
		rr := []rune(s)
		if len(rr) <= 8 {
			add(s)
		}
		if len(rr) > 2 {
			for i := 0; i+2 <= len(rr); i++ {
				add(string(rr[i : i+2]))
			}
		}
	}
	if len(out) > 18 {
		out = out[:18]
	}
	return out
}
func textScore(text string, terms []string) int {
	n := 0
	text = strings.ToLower(text)
	for _, t := range terms {
		if strings.Contains(text, t) {
			n += len([]rune(t))
		}
	}
	return n
}
func enrichPage(ctx context.Context, db *sql.DB, id string) (map[string]any, error) {
	if isClaudeLibrary(db) {
		return claudePage(ctx, db, id)
	}
	rows, e := rowsJSON(ctx, db, `SELECT p.*,v.teacher,v.institution,v.course FROM page p JOIN video v ON v.tag=p.tag WHERE p.id=?`, id)
	if e != nil {
		return nil, e
	}
	if len(rows) == 0 {
		return nil, errors.New("页面不存在")
	}
	p := rows[0]
	specs := []struct{ key, q string }{
		{"occurrences", `SELECT t0,t1 FROM page_occurrence WHERE page_id=? ORDER BY t0`},
		{"utterances", `SELECT DISTINCT u.id,u.version,u.t0,u.t1,u.text,u.quality_status,s.speaker_role,s.teacher_attribution FROM utterance u JOIN page_utterance pu ON pu.utterance_id=u.id LEFT JOIN utterance_speaker_scope s ON s.utterance_id=u.id WHERE pu.page_id=? ORDER BY u.t0,u.version LIMIT 500`},
		{"issues", `SELECT issue,action,status FROM review_issue WHERE page_id=?`},
		{"images", `SELECT asset_id,source_seconds,kind FROM page_image WHERE page_id=?`},
		{"official_pages", `SELECT s.id,s.pdf_page_no,s.direct_text,s.ocr_text,l.relation,l.notes,d.title,d.asset_id FROM page_source_link l JOIN source_document_page s ON s.id=l.source_page_id JOIN source_document d ON d.id=s.document_id WHERE l.page_id=?`},
		{"annotations", `SELECT DISTINCT t.original,t.candidate,t.basis,t.status,t.utterance_id FROM transcript_annotation t JOIN page_utterance pu ON pu.utterance_id=t.utterance_id WHERE pu.page_id=?`},
	}
	for _, s := range specs {
		x, e := rowsJSON(ctx, db, s.q, id)
		if e != nil {
			return nil, e
		}
		p[s.key] = x
	}
	var analysis string
	if e = db.QueryRowContext(ctx, "SELECT analysis_json FROM page_analysis WHERE page_id=?", id).Scan(&analysis); e == nil {
		var v any
		if json.Unmarshal([]byte(analysis), &v) == nil {
			p["analysis"] = v
		}
	}
	p["transcript_limit"] = 500
	p["reference"] = id
	return p, nil
}
func (a *App) libraryPage(w http.ResponseWriter, r *http.Request) {
	var x struct {
		ID string `json:"id"`
	}
	if !a.read(w, r, &x) {
		return
	}
	db, _, e := getLibrary()
	if e != nil {
		fail(w, 409, e.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	p, e := enrichPage(ctx, db, x.ID)
	if e != nil {
		fail(w, 404, e.Error())
		return
	}
	jsonOut(w, 200, p)
}
func (a *App) librarySearch(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	timings := map[string]int64{}
	last := started
	mark := func(name string) { now := time.Now(); timings[name] = now.Sub(last).Milliseconds(); last = now }
	var x struct {
		Q      string `json:"q"`
		Tag    string `json:"tag"`
		PageNo int    `json:"page_no"`
	}
	if !a.read(w, r, &x) {
		return
	}
	if len([]rune(x.Q)) > 2000 {
		fail(w, 400, "检索问题过长")
		return
	}
	db, _, e := getLibrary()
	if e != nil {
		jsonOut(w, 200, map[string]any{"pages": []any{}, "raw_ocr": []any{}, "official_hits": []any{}, "research_hits": []any{}, "document_hits": builtinSearch(r.Context(), x.Q), "timings": map[string]int64{"total_ms": time.Since(started).Milliseconds()}, "retrieval": "仅使用内置往届参考；未连接讲座资料库"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	vs, e := rowsJSON(ctx, db, "SELECT tag,teacher,institution FROM video")
	if e != nil {
		fail(w, 500, e.Error())
		return
	}
	named := map[string]bool{}
	for _, v := range vs {
		t, _ := v["teacher"].(string)
		if t != "" && strings.Contains(x.Q, t) {
			named[v["tag"].(string)] = true
		}
	}

	if strings.Contains(x.Q, "圆桌") {
		named["roundtable_future"] = true
	}

	if len(named) == 0 {
		for track, tag := range map[string]string{"新文科": "tangdan", "新医科": "xinye", "新农科": "xiangmiaolian", "新工科": "liuxueshen_xgk", "基础课程赛道": "zhangjing", "课程思政赛道": "caominhui", "产教融合赛道": "yumeifang", "实验教学赛道": "liuzhanhe"} {
			if strings.Contains(x.Q, track) {
				named[tag] = true
			}
		}
	}
	if len(named) == 0 && (strings.Contains(x.Q, "人工智能赛道") || strings.Contains(strings.ToLower(x.Q), "ai赛道")) {
		named["lijing"] = true
		named["songxiaofei"] = true
	}
	winnerQ := strings.Contains(x.Q, "九位") && strings.Contains(x.Q, "一等奖")
	if winnerQ {
		for _, tag := range []string{"tangdan", "yumeifang", "zhangjing", "liuzhanhe", "xiangmiaolian", "xinye", "liuxueshen_xgk", "caominhui", "lijing"} {
			named[tag] = true
		}
	}
	named = libraryTagHints(vs, named)
	q := `SELECT p.id,p.tag,p.page_no,p.title,p.page_text,v.teacher,v.course FROM page p JOIN video v ON p.tag=v.tag WHERE (?='' OR p.tag=?) AND (?=0 OR p.page_no=?)`
	ps, e := cachedTextRows(ctx, db, q, x.Tag, x.Tag, x.PageNo, x.PageNo)
	if e != nil {
		fail(w, 500, e.Error())
		return
	}
	terms := searchTerms(x.Q)
	type scored struct {
		p map[string]any
		s int
	}
	rank := []scored{}
	scoreMap := map[string]int{}
	// Search page facts first; retrieve oral-only matches without treating ASR as page text.
	if len(terms) > 0 && x.PageNo == 0 && (strings.Contains(x.Q, "原话") || strings.Contains(x.Q, "口述") || strings.Contains(x.Q, "说过") || strings.Contains(x.Q, "说了") || strings.Contains(x.Q, "讲过") || strings.Contains(x.Q, "讲了")) {
		parts := []string{}
		args := []any{x.Tag, x.Tag}
		for _, t := range terms {
			parts = append(parts, "instr(u.text,?)>0")
			args = append(args, t)
		}
		us, err := rowsJSON(ctx, db, `SELECT DISTINCT pu.page_id FROM utterance u JOIN page_utterance pu ON pu.utterance_id=u.id WHERE (?='' OR u.tag=?) AND (`+strings.Join(parts, " OR ")+`) LIMIT 1500`, args...)
		if err == nil {
			for _, u := range us {
				scoreMap[u["page_id"].(string)] = 2
			}
		}
	}
	for _, p := range ps {
		tag := p["tag"].(string)
		if len(named) > 0 && !named[tag] {
			continue
		}
		s := textScore(fmt.Sprint(p["title"]), terms)*4 + textScore(fmt.Sprint(p["page_text"]), terms) + scoreMap[p["id"].(string)]
		if x.PageNo > 0 || len(terms) == 0 || x.Q == "" {
			s += 1
		}
		if named[tag] {
			s += 2
		}
		if s > 0 {
			rank = append(rank, scored{p, s})
		}
	}
	sort.SliceStable(rank, func(i, j int) bool { return rank[i].s > rank[j].s })
	selected := []map[string]any{}
	perTag := map[string]int{}
	limit := 8
	if winnerQ {
		limit = 9
	}
	if x.Q == "" && x.Tag != "" {
		limit = 200
	}
	for _, s := range rank {
		tag := s.p["tag"].(string)
		capPerTag := 2
		if len(named) > 0 {
			capPerTag = 8 / len(named)
			if capPerTag < 1 {
				capPerTag = 1
			}
		}
		if x.Tag == "" && x.PageNo == 0 && perTag[tag] >= capPerTag {
			continue
		}
		selected = append(selected, s.p)
		perTag[tag]++
		if len(selected) >= limit {
			break
		}
	}
	mark("rank_ms")
	pages := []map[string]any{}
	for _, p := range selected {
		d, err := enrichPage(ctx, db, p["id"].(string))
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		pages = append(pages, d)
	}
	mark("page_details_ms")
	// Preserve raw OCR as a separately labeled evidence tier, including flash frames.
	raw := []map[string]any{}
	if len(terms) > 0 && x.Q != "" && (strings.Contains(x.Q, "快闪") || strings.Contains(strings.ToLower(x.Q), "ocr") || strings.Contains(x.Q, "闪现")) {
		parts := []string{}
		args := []any{x.Tag, x.Tag}
		strong := terms
		if len(strong) > 4 {
			strong = strong[:4]
		}
		for _, t := range strong {
			parts = append(parts, "instr(t.text,?)>0")
			args = append(args, t)
		}
		rr, err := rowsJSON(ctx, db, `SELECT o.tag,min(o.seconds) as seconds,t.id as text_id,t.text FROM ocr_observation o JOIN ocr_text_record t ON t.id=o.text_id WHERE (?='' OR o.tag=?) AND (`+strings.Join(parts, " OR ")+`) GROUP BY o.tag,t.id LIMIT 12`, args...)
		if err == nil {
			raw = rr
		}
	}
	if isClaudeLibrary(db) && (strings.Contains(x.Q, "快闪") || strings.Contains(strings.ToLower(x.Q), "ocr") || strings.Contains(x.Q, "闪现")) {
		raw = claudeRawOCR(ctx, db, x.Q, x.Tag)
	}
	mark("ocr_ms")
	official := []map[string]any{}
	research := []map[string]any{}
	if len(terms) > 0 {
		strong := terms
		if len(strong) > 6 {
			strong = strong[:6]
		}
		parts := []string{}
		args := []any{x.Tag, x.Tag}
		for _, t := range strong {
			parts = append(parts, "instr(s.direct_text||s.ocr_text,?)>0")
			args = append(args, t)
		}
		hits, err := rowsJSON(ctx, db, `SELECT s.id,s.pdf_page_no,s.direct_text,s.ocr_text,d.tag,d.title,d.asset_id,v.teacher FROM source_document_page s JOIN source_document d ON d.id=s.document_id LEFT JOIN video v ON v.tag=d.tag WHERE (?='' OR d.tag=?) AND (`+strings.Join(parts, " OR ")+`) LIMIT 12`, args...)
		if err == nil {
			for _, h := range hits {
				if len(named) == 0 || named[fmt.Sprint(h["tag"])] {
					official = append(official, h)
				}
			}
		}
		parts = []string{}
		args = []any{}
		for _, t := range strong {
			parts = append(parts, "instr(title||payload_json,?)>0")
			args = append(args, t)
		}
		hits, err = rowsJSON(ctx, db, `SELECT id,kind,title,payload_json FROM research_card WHERE `+strings.Join(parts, " OR ")+` LIMIT 6`, args...)
		if err == nil {
			research = hits
		}
	}
	mark("references_ms")
	documents := knowledgeSearch(ctx, db, x.Q, x.Tag)
	mark("reports_ms")
	timings["total_ms"] = time.Since(started).Milliseconds()
	jsonOut(w, 200, map[string]any{"timings": timings, "pages": pages, "raw_ocr": raw, "official_hits": official, "research_hits": research, "document_hits": append(builtinSearch(ctx, x.Q), documents...), "matched_pages": len(rank), "returned_pages": len(pages), "retrieval": "关键词检索；跨场结果保留多专家来源。研究卡为研究判断，不是原话；原始转写与OCR未视为听校金标准。"})
}
func (a *App) libraryAsset(w http.ResponseWriter, r *http.Request) {
	var x struct {
		ID string `json:"id"`
	}
	if !a.read(w, r, &x) {
		return
	}
	db, _, e := getLibrary()
	if e != nil {
		fail(w, 409, e.Error())
		return
	}
	var b []byte
	var mime string
	if isClaudeLibrary(db) {
		mime, b, e = claudeAsset(r.Context(), db, x.ID)
	} else {
		e = db.QueryRowContext(r.Context(), "SELECT mime,content FROM asset WHERE id=?", x.ID).Scan(&mime, &b)
	}
	if e != nil {
		fail(w, 404, "媒体不存在")
		return
	}
	b, e = decodeMedia(r.Context(), db, "asset", x.ID, b)
	if e != nil {
		fail(w, 500, e.Error())
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Length", fmt.Sprint(len(b)))
	_, _ = w.Write(b)
}
func (a *App) libraryAudio(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Tag  string  `json:"tag"`
		Time float64 `json:"time"`
	}
	if !a.read(w, r, &x) {
		return
	}
	db, _, e := getLibrary()
	if e != nil {
		fail(w, 409, e.Error())
		return
	}
	if isClaudeLibrary(db) {
		claudeAudio(w, r, db, x.Tag, x.Time)
		return
	}
	rows, e := rowsJSON(r.Context(), db, `SELECT id,asset_id,t0,t1,source_audio_start,encoded_duration,quality_status FROM audio_chunk WHERE tag=? AND t0<=? AND t1>? ORDER BY t0 DESC LIMIT 1`, x.Tag, x.Time, x.Time)
	if e != nil || len(rows) == 0 {
		fail(w, 404, "该时刻没有可用音频块")
		return
	}
	jsonOut(w, 200, rows[0])
}

// Used by integration tests to ensure all read paths remain read-only.
func (a *App) libraryCatalog(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Tag string `json:"tag"`
	}
	if !a.read(w, r, &x) {
		return
	}
	db, _, e := getLibrary()
	if e != nil {
		fail(w, 409, e.Error())
		return
	}
	rows, e := rowsJSON(r.Context(), db, `SELECT p.id,p.tag,p.page_no,p.title,v.teacher FROM page p JOIN video v ON v.tag=p.tag WHERE (?='' OR p.tag=?) ORDER BY v.teacher,p.page_no`, x.Tag, x.Tag)
	if e != nil {
		fail(w, 500, e.Error())
		return
	}
	jsonOut(w, 200, rows)
}

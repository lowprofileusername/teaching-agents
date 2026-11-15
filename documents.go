package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

func knowledgeSearch(ctx context.Context, db *sql.DB, q, tag string) []map[string]any {
	out := []map[string]any{}
	rows, e := cachedTextRows(ctx, db, `SELECT p.id,p.pdf_page_no,p.text,p.search_text,p.extraction,d.id as document_id,d.title,d.kind,d.video_tag,d.edition,d.authority_level,d.authority_status,d.source_url FROM knowledge_document_page p JOIN knowledge_document d ON d.id=p.document_id WHERE (?='' OR d.video_tag=? OR d.kind='competition_rule' OR d.video_tag IS NULL)`, tag, tag)
	if e != nil {
		return out
	}
	terms := searchTerms(q)
	type candidate struct {
		r map[string]any
		s int
	}
	rank := []candidate{}

	ruleQ := strings.Contains(q, "评分") || strings.Contains(q, "国赛正式") || strings.Contains(q, "规则") || strings.Contains(q, "校赛")
	crossQ := false
	for _, term := range []string{"九位", "共同", "综合", "36场", "四层", "八种", "图谱", "创新报告应该怎么写", "报告写作"} {
		if strings.Contains(q, term) {
			crossQ = true
		}
	}
	named := map[string]bool{}
	vs, _ := cachedTextRows(ctx, db, "SELECT tag,teacher FROM video")
	for _, v := range vs {
		teacher := fmt.Sprint(v["teacher"])
		if teacher != "" && strings.Contains(q, teacher) {
			named[fmt.Sprint(v["tag"])] = true
		}
	}
	if strings.Contains(q, "圆桌") {
		named["roundtable_future"] = true
	}
	allowProposal := strings.Contains(q, "旅游电子商务") || strings.Contains(q, "我的") || strings.Contains(q, "我上传")

	for _, r := range rows {
		if r["kind"] == "user_material_proposal" && !allowProposal {
			continue
		}
		score := textScore(fmt.Sprint(r["title"]), terms)*3 + textScore(fmt.Sprint(r["search_text"]), terms)

		if r["kind"] == "competition_rule" {
			if ruleQ {
				score += 50
			} else {
				score -= 30
			}
		}
		if crossQ && (r["kind"] == "research_report" || r["kind"] == "research_chart") {
			score += 35
		}
		if len(named) > 0 {
			if named[fmt.Sprint(r["video_tag"])] {
				score += 70
			} else if r["kind"] == "single_session_report" {
				continue
			}
		}

		if score > 0 {
			rank = append(rank, candidate{r, score})
		}
	}
	sort.SliceStable(rank, func(i, j int) bool { return rank[i].s > rank[j].s })

	per := map[string]int{}
	covered := map[string]bool{}
	// For a named comparison, reserve one report from each requested speaker.
	for _, v := range rank {
		tag := fmt.Sprint(v.r["video_tag"])
		if !named[tag] || covered[tag] {
			continue
		}
		id := fmt.Sprint(v.r["document_id"])
		v.r["text"] = stringLimit(fmt.Sprint(v.r["text"]), 4500)
		delete(v.r, "search_text")
		v.r["citation"] = fmt.Sprintf("[%s PDF第%v页]", v.r["title"], v.r["pdf_page_no"])
		out = append(out, v.r)
		per[id]++
		covered[tag] = true
		if len(out) >= 6 {
			return out
		}
	}

	for _, v := range rank {
		duplicate := false
		for _, o := range out {
			if o["id"] == v.r["id"] {
				duplicate = true
			}
		}
		if duplicate {
			continue
		}
		id := fmt.Sprint(v.r["document_id"])
		if per[id] >= 2 {
			continue
		}
		v.r["text"] = stringLimit(fmt.Sprint(v.r["text"]), 4500)
		delete(v.r, "search_text")
		v.r["citation"] = fmt.Sprintf("[%s PDF第%v页]", v.r["title"], v.r["pdf_page_no"])
		out = append(out, v.r)
		per[id]++
		if len(out) == 6 {
			break
		}
	}
	return out
}
func stringLimit(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "\n〔仅提供本页前部，未展示全文〕"
	}
	return s
}
func (a *App) documents(w http.ResponseWriter, r *http.Request) {
	var x struct{}
	if !a.read(w, r, &x) {
		return
	}
	db, _, e := getLibrary()
	if e != nil {
		fail(w, 409, e.Error())
		return
	}
	rows, e := rowsJSON(r.Context(), db, `SELECT id,title,kind,video_tag,pdf_page_count,edition,authority_level,authority_status FROM knowledge_document ORDER BY kind,title`)
	if e != nil {
		jsonOut(w, 200, map[string]any{"documents": []any{}, "note": "此数据库尚未添加报告与规则扩展"})
		return
	}
	rules, _ := rowsJSON(r.Context(), db, `SELECT * FROM competition_track_status`)
	jsonOut(w, 200, map[string]any{"documents": rows, "rule_status": rules})
}

// Original PDFs are opened only through their recorded database identity, never an arbitrary supplied path.
func (a *App) documentPDF(w http.ResponseWriter, r *http.Request) {
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
	// Portable libraries contain complete PDFs. This media table is never joined during search.
	var embedded []byte
	var embeddedSHA string
	err := db.QueryRowContext(r.Context(), `SELECT content,sha256 FROM knowledge_document_pdf WHERE document_id=?`, x.ID).Scan(&embedded, &embeddedSHA)
	if err == nil {
		embedded, err = decodeMedia(r.Context(), db, "knowledge_document_pdf", x.ID, embedded)
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		sum := sha256.Sum256(embedded)
		if hex.EncodeToString(sum[:]) != embeddedSHA {
			fail(w, 500, "库内PDF校验失败")
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Length", fmt.Sprint(len(embedded)))
		_, _ = w.Write(embedded)
		return
	}
	if r.Context().Err() != nil {
		fail(w, 408, "读取PDF已取消或超时")
		return
	}
	var source, sha string
	if db.QueryRowContext(r.Context(), `SELECT source_path,sha256 FROM knowledge_document WHERE id=?`, x.ID).Scan(&source, &sha) != nil {
		fail(w, 404, "报告不存在")
		return
	}
	source = platformDocumentPath(source)
	f, e := os.Open(source)
	if e != nil {
		fail(w, 404, "原PDF未找到，请连接原共享盘")
		return
	}
	defer f.Close()
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil {
		fail(w, 500, "PDF读取失败")
		return
	}
	if hex.EncodeToString(h.Sum(nil)) != sha {
		fail(w, 409, "PDF已更新，与入库版本不一致；请重新入库后查看")
		return
	}
	_, _ = f.Seek(0, 0)
	info, e := f.Stat()
	if e != nil {
		fail(w, 500, "PDF状态读取失败")
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}
func platformDocumentPath(p string) string {
	_, dbPath, e := getLibrary()
	if e != nil {
		return p
	}
	return resolveDocumentPath(p, dbPath)
}
func resolveDocumentPath(p, dbPath string) string {
	slash := strings.ReplaceAll(dbPath, "\\", "/")
	source := strings.ReplaceAll(p, "\\", "/")
	marker := "/" + projectFolder + "/"
	j := strings.Index(source, marker)
	candidates := []string{}
	if j >= 0 {
		rel := source[j+len(marker):]
		// A travel copy keeps report/... and codex_analysis_tmp/... alongside the DB.
		candidates = append(candidates, filepath.Join(filepath.Dir(dbPath), filepath.FromSlash(rel)))
		if i := strings.Index(slash, "/codex_analysis_tmp/"); i >= 0 {
			candidates = append(candidates, slash[:i]+"/"+rel)
		}
	}
	candidates = append(candidates, p)
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return p
}

type MaterialUnit struct {
	No    int    `json:"no"`
	Label string `json:"label"`
	Text  string `json:"text"`
}

func xmlText(data []byte) (string, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	var out strings.Builder
	inText := false
	for {
		t, e := d.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return "", e
		}
		switch v := t.(type) {
		case xml.StartElement:
			if v.Name.Local == "t" {
				inText = true
			}
			if v.Name.Local == "tab" {
				out.WriteByte('\t')
			}
		case xml.CharData:
			if inText {
				out.Write(v)
			}
		case xml.EndElement:
			if v.Name.Local == "t" {
				inText = false
			}
			if v.Name.Local == "p" {
				out.WriteByte('\n')
			}
		}
	}
	return out.String(), nil
}
func parseOffice(data []byte, ext string) ([]MaterialUnit, error) {
	zr, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil {
		return nil, errors.New("不是有效的 Office 文件")
	}
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	var used int64
	read := func(name string) ([]byte, error) {
		f := files[name]
		if f == nil {
			return nil, errors.New("缺少文件结构：" + name)
		}
		if f.UncompressedSize64 > 8<<20 {
			return nil, errors.New("单个文档部件过大")
		}
		rr, e := f.Open()
		if e != nil {
			return nil, e
		}
		defer rr.Close()
		b, e := io.ReadAll(io.LimitReader(rr, (8<<20)+1))
		used += int64(len(b))
		if len(b) > 8<<20 || used > 32<<20 {
			return nil, errors.New("解压文字超出限制")
		}
		return b, e
	}
	units := []MaterialUnit{}
	if ext == ".docx" {
		b, e := read("word/document.xml")
		if e != nil {
			return nil, e
		}
		s, e := xmlText(b)
		if e != nil {
			return nil, e
		}
		paras := strings.Split(s, "\n")
		var part strings.Builder
		start := 1
		flush := func(end int) {
			if strings.TrimSpace(part.String()) != "" {
				units = append(units, MaterialUnit{len(units) + 1, fmt.Sprintf("段落 %d–%d（非排版页码）", start, end), part.String()})
			}
			part.Reset()
			start = end + 1
		}
		for i, p := range paras {
			part.WriteString(p + "\n")
			if part.Len() > 5000 {
				flush(i + 1)
			}
		}
		flush(len(paras))
		return units, nil
	}
	if ext != ".pptx" {
		return nil, errors.New("支持 PDF、DOCX、PPTX、TXT、MD；旧版 DOC/PPT 请另存为新格式")
	}
	pres, e := read("ppt/presentation.xml")
	if e != nil {
		return nil, e
	}
	rels, e := read("ppt/_rels/presentation.xml.rels")
	if e != nil {
		return nil, e
	}
	relmap := map[string]string{}
	d := xml.NewDecoder(bytes.NewReader(rels))
	for {
		t, e := d.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, e
		}
		if v, ok := t.(xml.StartElement); ok && v.Name.Local == "Relationship" {
			id, target, mode := "", "", ""
			for _, a := range v.Attr {
				switch a.Name.Local {
				case "Id":
					id = a.Value
				case "Target":
					target = a.Value
				case "TargetMode":
					mode = a.Value
				}
			}
			if mode != "External" {
				if strings.HasPrefix(target, "/") {
					relmap[id] = path.Clean(strings.TrimPrefix(target, "/"))
				} else {
					relmap[id] = path.Clean(path.Join("ppt", target))
				}
			}
		}
	}
	d = xml.NewDecoder(bytes.NewReader(pres))
	for {
		t, e := d.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, e
		}
		if v, ok := t.(xml.StartElement); ok && v.Name.Local == "sldId" {
			rid := ""
			for _, a := range v.Attr {
				if a.Name.Local == "id" && strings.Contains(a.Name.Space, "relationships") {
					rid = a.Value
				}
			}
			target := relmap[rid]
			if !strings.HasPrefix(target, "ppt/slides/") {
				return nil, errors.New("幻灯片引用无效")
			}
			b, e := read(target)
			if e != nil {
				return nil, e
			}
			s, e := xmlText(b)
			if e != nil {
				return nil, e
			}
			n := len(units) + 1
			units = append(units, MaterialUnit{n, fmt.Sprintf("幻灯片第%d页", n), s})
			if n > 300 {
				return nil, errors.New("单次最多300页")
			}
		}
	}
	return units, nil
}
func (a *App) parseMaterial(w http.ResponseWriter, r *http.Request) {
	name, e := url.QueryUnescape(r.Header.Get("X-File-Name"))
	if e != nil {
		fail(w, 400, "文件名无效")
		return
	}
	b, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 50<<20))
	if e != nil {
		fail(w, 413, "单文件不能超过50 MB")
		return
	}
	units, e := parseOffice(b, strings.ToLower(path.Ext(name)))
	if e != nil {
		fail(w, 400, e.Error())
		return
	}
	empty := 0
	for _, u := range units {
		if strings.TrimFunc(u.Text, unicode.IsSpace) == "" {
			empty++
		}
	}
	jsonOut(w, 200, map[string]any{"name": name, "units": units, "empty_units": empty, "scope": "原生文字提取；未读取图片中的字、SmartArt、母版文字、备注、图表曲线或版式；Word按段落组，PPTX按实际幻灯片顺序。文件仅在内存解析，不保存原文件。"})
}

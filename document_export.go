package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ParagraphEdit struct {
	ID          string `json:"paragraph_id"`
	Original    string `json:"original"`
	Comment     string `json:"comment"`
	Replacement string `json:"replacement"`
}
type reviewExportResult struct {
	Comments  int      `json:"comments"`
	Revisions int      `json:"revisions"`
	Warnings  []string `json:"warnings"`
}

func xmlEsc(s string) string { var b bytes.Buffer; xml.EscapeText(&b, []byte(s)); return b.String() }
func appendXML(b []byte, content string) ([]byte, error) {
	n, e := xmlSpans(b)
	if e != nil {
		return nil, e
	}
	if n.CloseStart == n.End {
		head := string(b[:n.OpenEnd])
		i := strings.LastIndex(head, "/>")
		if i < 0 {
			return nil, errors.New("不能扩展XML元素")
		}
		name := strings.Fields(strings.TrimPrefix(string(b[n.Start:n.OpenEnd]), "<"))[0]
		name = strings.TrimSuffix(name, "/>")
		return []byte(head[:i] + ">" + content + "</" + name + ">" + string(b[n.End:])), nil
	}
	return []byte(string(b[:n.CloseStart]) + content + string(b[n.CloseStart:])), nil
}
func ensureRel(parts map[string][]byte, typ, target string) error {
	key := "word/_rels/document.xml.rels"
	b := parts[key]
	if len(b) == 0 {
		b = []byte(`<Relationships xmlns="` + relNS + `"></Relationships>`)
	}
	root, e := xmlSpans(b)
	if e != nil {
		return e
	}
	ids := map[string]bool{}
	for _, c := range root.Children {
		ids[attr(c, "Id")] = true
		if attr(c, "Type") == officeRelNS+typ {
			return nil
		}
	}
	i := 1
	for ids[fmt.Sprintf("rIdReview%d", i)] {
		i++
	}
	parts[key], e = appendXML(b, fmt.Sprintf(`<Relationship xmlns="%s" Id="rIdReview%d" Type="%s%s" Target="%s"/>`, relNS, i, officeRelNS, typ, xmlEsc(target)))
	return e
}
func ensureContent(parts map[string][]byte, part, kind string) error {
	b := parts["[Content_Types].xml"]
	root, e := xmlSpans(b)
	if e != nil {
		return e
	}
	for _, c := range root.Children {
		if attr(c, "PartName") == "/"+part {
			return nil
		}
	}
	parts["[Content_Types].xml"], e = appendXML(b, `<Override xmlns="http://schemas.openxmlformats.org/package/2006/content-types" PartName="/`+xmlEsc(part)+`" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.`+kind+`+xml"/>`)
	return e
}
func nextXMLID(parts map[string][]byte) int {
	max := 0
	for name, b := range parts {
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		root, e := xmlSpans(b)
		if e != nil {
			continue
		}
		walkXML(root, func(n *XMLSpan) {
			for _, a := range n.Attr {
				if a.Name.Local == "id" {
					if i, e := strconv.Atoi(a.Value); e == nil && i > max && i < 1000000000 {
						max = i
					}
				}
			}
		})
	}
	return max + 1
}
func replaceTextTags(raw []byte) ([]byte, error) {
	root, e := xmlSpans(raw)
	if e != nil {
		return nil, e
	}
	type patch struct {
		s, e int
		v    string
	}
	ps := []patch{}
	walkXML(root, func(n *XMLSpan) {
		if n.Name.Local == "t" {
			opening := string(raw[n.Start:n.OpenEnd])
			k := strings.IndexAny(opening, " >/")
			if k < 0 {
				return
			}
			name := opening[1:k]
			prefix := ""
			if at := strings.LastIndex(name, ":"); at >= 0 {
				prefix = name[:at+1]
			}
			ps = append(ps, patch{n.Start + 1, n.Start + k, prefix + "delText"})
			if n.CloseStart < n.End {
				ps = append(ps, patch{n.CloseStart + 2, n.End - 1, prefix + "delText"})
			}
		}
	})
	sort.Slice(ps, func(i, j int) bool { return ps[i].s > ps[j].s })
	for _, p := range ps {
		raw = append(append(append([]byte{}, raw[:p.s]...), []byte(p.v)...), raw[p.e:]...)
	}
	return raw, nil
}
func textRun(s, rPr string) string {
	var b strings.Builder
	b.WriteString("<w:r>" + rPr)
	for i, line := range strings.Split(s, "\n") {
		if i > 0 {
			b.WriteString("<w:br/>")
		}
		for j, segment := range strings.Split(line, "\t") {
			if j > 0 {
				b.WriteString("<w:tab/>")
			}
			b.WriteString(`<w:t xml:space="preserve">` + xmlEsc(segment) + `</w:t>`)
		}
	}
	b.WriteString("</w:r>")
	return b.String()
}
func revisionParagraph(raw []byte, p *XMLSpan, replacement string, id int, date string) ([]byte, bool, error) {
	// Complex fields, drawings, existing revisions, hyperlinks, nested textboxes,
	// and anchored ranges remain intact: the suggestion goes into a comment.
	start, end := -1, -1
	rPr := ""
	for _, c := range p.Children {
		if c.Name.Local == "pPr" {
			continue
		}
		if c.Name.Space != wordNS || c.Name.Local != "r" {
			return nil, false, nil
		}
		for _, v := range c.Children {
			if v.Name.Space != wordNS || !oneOf(v.Name.Local, "rPr", "t", "tab", "br") {
				return nil, false, nil
			}
			if v.Name.Local == "rPr" && start < 0 {
				rPr = string(raw[v.Start:v.End])
			}
		}
		if start < 0 {
			start = c.Start
		}
		end = c.End
	}
	if start < 0 {
		return nil, false, nil
	}
	deleted, e := replaceTextTags([]byte(`<root xmlns:w="` + wordNS + `">` + string(raw[start:end]) + `</root>`))
	if e != nil {
		return nil, false, e
	}
	root, e := xmlSpans(deleted)
	if e != nil {
		return nil, false, e
	}
	del := string(deleted[root.OpenEnd:root.CloseStart])
	ns := ` xmlns:w="` + wordNS + `"`
	attrs := ` w:author="教创赛专家" w:date="` + date + `"`
	change := fmt.Sprintf(`<w:del%s w:id="%d"%s>%s</w:del><w:ins%s w:id="%d"%s>%s</w:ins>`, ns, id, attrs, del, ns, id+1, attrs, textRun(replacement, rPr))
	out := string(raw[p.Start:start]) + change + string(raw[end:p.End])
	return []byte(out), true, nil
}
func addCommentAnchor(raw []byte, id int) ([]byte, error) {
	p, e := xmlSpans(raw)
	if e != nil {
		return nil, e
	}
	ns := ` xmlns:w="` + wordNS + `"`
	start := fmt.Sprintf(`<w:commentRangeStart%s w:id="%d"/>`, ns, id)
	end := fmt.Sprintf(`<w:commentRangeEnd%s w:id="%d"/><w:r%s><w:rPr><w:rStyle w:val="CommentReference"/></w:rPr><w:commentReference w:id="%d"/></w:r>`, ns, id, ns, id)
	if p.CloseStart == p.End {
		return appendXML(raw, start+end)
	}
	at := p.OpenEnd
	for _, c := range p.Children {
		if c.Name.Local == "pPr" {
			at = c.End
			break
		}
	}
	return []byte(string(raw[:at]) + start + string(raw[at:p.CloseStart]) + end + string(raw[p.CloseStart:])), nil
}
func buildReviewedDOCX(data []byte, edits []ParagraphEdit, mode string) ([]byte, reviewExportResult, error) {
	result := reviewExportResult{Warnings: []string{}}
	if !oneOf(mode, "comments", "revisions") || len(edits) == 0 || len(edits) > 3000 {
		return nil, result, errors.New("请选择批注或修订，并先完成审阅")
	}
	parts, e := officeParts(data)
	if e != nil {
		return nil, result, e
	}
	for k := range parts {
		if strings.HasPrefix(k, "_xmlsignatures/") || strings.HasSuffix(k, "vbaProject.bin") {
			return nil, result, errors.New("不改写带数字签名或宏的文档，请另存为普通DOCX")
		}
	}
	ps, e := docxParagraphs(data)
	if e != nil {
		return nil, result, e
	}
	texts := map[string]string{}
	for _, p := range ps {
		texts[p.ID] = p.Text
	}
	doc := parts["word/document.xml"]
	root, e := xmlSpans(doc)
	if e != nil {
		return nil, result, e
	}
	spans := map[string]*XMLSpan{}
	pi := 0
	walkXML(root, func(n *XMLSpan) {
		if n.Name.Space == wordNS && n.Name.Local == "p" {
			pi++
			spans[fmt.Sprintf("p%d", pi)] = n
		}
	})
	commentsPart := "word/comments.xml"
	rels := parts["word/_rels/document.xml.rels"]
	if len(rels) > 0 {
		rr, e := xmlSpans(rels)
		if e != nil {
			return nil, result, e
		}
		for _, n := range rr.Children {
			if attr(n, "Type") == officeRelNS+"comments" {
				commentsPart = partRelationships(parts, "word/document.xml")[attr(n, "Id")]
				if !strings.HasPrefix(commentsPart, "word/") {
					return nil, result, errors.New("批注关系无效")
				}
			}
		}
	}
	comments := parts[commentsPart]
	if len(comments) == 0 {
		comments = []byte(`<?xml version="1.0" encoding="UTF-8"?><w:comments xmlns:w="` + wordNS + `"></w:comments>`)
	}
	next := nextXMLID(parts)
	date := time.Now().UTC().Format(time.RFC3339)
	type patch struct {
		s, e int
		b    []byte
	}
	patches := []patch{}
	seen := map[string]bool{}
	for _, edit := range edits {
		p := spans[edit.ID]
		original, ok := texts[edit.ID]
		if !ok || p == nil || original != edit.Original || seen[edit.ID] {
			return nil, result, fmt.Errorf("%s 原文/编号不匹配或重复，未写入文件", edit.ID)
		}
		seen[edit.ID] = true
		if len(edit.Comment) > 100000 || len(edit.Replacement) > 200000 || strings.TrimSpace(edit.Comment) == "" {
			return nil, result, errors.New("批注内容为空或超限")
		}
		nested := false
		for _, c := range p.Children {
			walkXML(c, func(n *XMLSpan) {
				if n.Name.Local == "p" && n.Name.Space == wordNS {
					nested = true
				}
			})
		}
		if nested {
			result.Warnings = append(result.Warnings, edit.ID+"包含嵌套文本框，保留在工作台，未写回此条")
			continue
		}
		content := append([]byte{}, doc[p.Start:p.End]...)
		comment := edit.Comment
		if edit.Replacement != "" && edit.Replacement != edit.Original {
			if mode == "revisions" {
				v, done, e := revisionParagraph(doc, p, edit.Replacement, next, date)
				if e != nil {
					return nil, result, e
				}
				if done {
					content = v
					next += 2
					result.Revisions++
				} else {
					result.Warnings = append(result.Warnings, edit.ID+"包含图片、域、链接、已有批注/修订或复杂对象，仅添加建议批注，不改正文")
				}
			}
			comment += "\n建议替换：\n" + edit.Replacement
		}
		cid := next
		next++
		content, e = addCommentAnchor(content, cid)
		if e != nil {
			return nil, result, e
		}
		patches = append(patches, patch{p.Start, p.End, content})
		entry := fmt.Sprintf(`<w:comment xmlns:w="%s" w:id="%d" w:author="教创赛专家" w:initials="教创" w:date="%s"><w:p>%s</w:p></w:comment>`, wordNS, cid, date, textRun(comment, ""))
		comments, e = appendXML(comments, entry)
		if e != nil {
			return nil, result, e
		}
		result.Comments++
	}
	if len(patches) == 0 {
		return nil, result, errors.New("没有可安全写回的批注；请查看复杂对象提示")
	}
	sort.Slice(patches, func(i, j int) bool { return patches[i].s > patches[j].s })
	last := len(doc)
	for _, p := range patches {
		if p.e > last {
			return nil, result, errors.New("审阅锚点存在重叠，未改写")
		}
		doc = append(append(append([]byte{}, doc[:p.s]...), p.b...), doc[p.e:]...)
		last = p.s
	}
	parts["word/document.xml"] = doc
	parts[commentsPart] = comments
	if e = ensureRel(parts, "comments", strings.TrimPrefix(commentsPart, "word/")); e != nil {
		return nil, result, e
	}
	if e = ensureContent(parts, commentsPart, "comments"); e != nil {
		return nil, result, e
	}
	if result.Revisions > 0 {
		settingsPart := "word/settings.xml"
		if len(rels) > 0 {
			rr, _ := xmlSpans(rels)
			for _, n := range rr.Children {
				if attr(n, "Type") == officeRelNS+"settings" {
					settingsPart = partRelationships(parts, "word/document.xml")[attr(n, "Id")]
					if !strings.HasPrefix(settingsPart, "word/") {
						return nil, result, errors.New("设置关系无效")
					}
				}
			}
		}
		settings := parts[settingsPart]
		if len(settings) == 0 {
			settings = []byte(`<w:settings xmlns:w="` + wordNS + `"></w:settings>`)
		}
		sr, e := xmlSpans(settings)
		if e != nil {
			return nil, result, e
		}
		has := false
		walkXML(sr, func(n *XMLSpan) {
			if n.Name.Local == "trackRevisions" {
				has = true
			}
		})
		if !has {
			settings, e = appendXML(settings, `<w:trackRevisions xmlns:w="`+wordNS+`"/>`)
			if e != nil {
				return nil, result, e
			}
		}
		parts[settingsPart] = settings
		if e = ensureRel(parts, "settings", strings.TrimPrefix(settingsPart, "word/")); e != nil {
			return nil, result, e
		}
		if e = ensureContent(parts, settingsPart, "settings"); e != nil {
			return nil, result, e
		}
	}
	zr, _ := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, f := range zr.File {
		b, ok := parts[f.Name]
		if !ok {
			continue
		}
		r, e := f.Open()
		if e != nil {
			return nil, result, e
		}
		old, e := io.ReadAll(r)
		r.Close()
		if e != nil {
			return nil, result, e
		}
		if bytes.Equal(old, b) {
			e = zw.Copy(f)
		} else {
			var w io.Writer
			w, e = zw.CreateHeader(&f.FileHeader)
			if e == nil {
				_, e = w.Write(b)
			}
		}
		if e != nil {
			return nil, result, e
		}
		delete(parts, f.Name)
	}
	names := []string{}
	for n := range parts {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		w, e := zw.Create(n)
		if e != nil {
			return nil, result, e
		}
		if _, e = w.Write(parts[n]); e != nil {
			return nil, result, e
		}
	}
	if e = zw.Close(); e != nil {
		return nil, result, e
	}
	return out.Bytes(), result, nil
}
func (a *App) exportDocumentReview(w http.ResponseWriter, r *http.Request) {
	var x struct {
		SourceID  string          `json:"source_id"`
		Name      string          `json:"name"`
		Directory string          `json:"directory"`
		Mode      string          `json:"mode"`
		Edits     []ParagraphEdit `json:"edits"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&x) != nil {
		fail(w, 400, "导出请求无效或超限")
		return
	}
	b, e := a.readSource(x.SourceID, ".docx")
	if e != nil {
		fail(w, 400, e.Error())
		return
	}
	out, result, e := buildReviewedDOCX(b, x.Edits, x.Mode)
	if e != nil {
		fail(w, 400, e.Error())
		return
	}
	dir := filepath.Clean(x.Directory)
	if !filepath.IsAbs(dir) {
		fail(w, 400, "请输入完整导出文件夹路径")
		return
	}
	if info, e := os.Stat(dir); e != nil {
		parent, pe := os.Stat(filepath.Dir(dir))
		if pe != nil || !parent.IsDir() {
			fail(w, 400, "父文件夹不存在，请检查共享盘挂载")
			return
		}
		if e = os.Mkdir(dir, 0700); e != nil && !os.IsExist(e) {
			fail(w, 400, e.Error())
			return
		}
	} else if !info.IsDir() {
		fail(w, 400, "导出位置必须为文件夹")
		return
	}
	label := "批注"
	if x.Mode == "revisions" {
		label = "修订"
	}
	nonce := make([]byte, 4)
	if _, e = rand.Read(nonce); e != nil {
		fail(w, 500, e.Error())
		return
	}
	name := strings.TrimSuffix(path.Base(strings.ReplaceAll(x.Name, "\\", "/")), ".docx")
	name = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, r) || r < 32 {
			return '_'
		}
		return r
	}, name)
	name = stringLimit(name, 70)
	dest := filepath.Join(dir, fmt.Sprintf("%s_%s_%s_%s.docx", name, label, time.Now().Format("20060102_150405"), hex.EncodeToString(nonce)))
	f, e := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		fail(w, 400, e.Error())
		return
	}
	_, e = f.Write(out)
	ce := f.Close()
	if e != nil || ce != nil {
		os.Remove(dest)
		fail(w, 500, "写入失败，未保留不完整文件")
		return
	}
	jsonOut(w, 200, map[string]any{"path": dest, "bytes": len(out), "result": result, "author": "教创赛专家"})
}

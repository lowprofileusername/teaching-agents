package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// RevisionSpan is a character-level (rune offset) tracked change inside one
// paragraph. Delete must equal Original[Start:End]; Insert replaces it.
// Pure insertion: Start==End and Delete=="". Pure deletion: Insert=="".
type RevisionSpan struct {
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Delete string `json:"delete"`
	Insert string `json:"insert"`
	Reason string `json:"reason,omitempty"`
}

func xmlUnescape(s string) string {
	d := xml.NewDecoder(strings.NewReader("<d>" + s + "</d>"))
	var b strings.Builder
	for {
		t, e := d.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return s
		}
		if c, ok := t.(xml.CharData); ok {
			b.Write(c)
		}
	}
	return b.String()
}

func validateSpans(original string, spans []RevisionSpan) error {
	if len(spans) > 200 {
		return errors.New("单段修订过多")
	}
	r := []rune(original)
	sorted := append([]RevisionSpan(nil), spans...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Start < sorted[j].Start })
	prevEnd := 0
	for _, s := range sorted {
		if s.Start < 0 || s.End < s.Start || s.End > len(r) {
			return errors.New("修订区间越界")
		}
		if s.Start < prevEnd {
			return errors.New("修订区间重叠")
		}
		if string(r[s.Start:s.End]) != s.Delete {
			return errors.New("修订delete与原文不一致")
		}
		if s.Delete == "" && s.Insert == "" {
			return errors.New("空修订")
		}
		if len([]rune(s.Insert)) > 4000 {
			return errors.New("修订insert超限")
		}
		if generatedFactRisk.MatchString(s.Insert) {
			return errors.New("修订insert含数字/引语/成效断言，涉嫌虚构")
		}
		prevEnd = s.End
	}
	return nil
}

func spanDetail(spans []RevisionSpan) string {
	var b strings.Builder
	for i, s := range spans {
		if i >= 50 {
			b.WriteString(fmt.Sprintf("…等共%d条\n", len(spans)))
			break
		}
		if s.Delete != "" {
			b.WriteString("删「" + s.Delete + "」")
		}
		if s.Insert != "" {
			b.WriteString("→ 插「" + s.Insert + "」")
		}
		if s.Reason != "" {
			b.WriteString("（" + s.Reason + "）")
		}
		b.WriteString("\n")
	}
	out := b.String()
	if r := []rune(out); len(r) > 20000 {
		out = string(r[:20000])
	}
	return out
}

type spanToken struct {
	start, end int
	kind       string // text | tab | br
	text       string // unescaped content for text tokens
	raw        string // original XML for tab/br tokens
	rPr        string
}

// revisionParagraphSpans writes fine-grained w:del/w:ins for the given spans,
// preserving untouched runs and their formatting. Returns (patched paragraph,
// ids consumed, ok, err). ok=false means the paragraph contains complex objects
// and the caller should fall back to a comment (or whole-paragraph revision).
func revisionParagraphSpans(raw []byte, p *XMLSpan, original string, spans []RevisionSpan, id int, date string) ([]byte, int, bool, error) {
	start, end := -1, -1
	var tokens []spanToken
	pos := 0
	for _, c := range p.Children {
		if c.Name.Local == "pPr" {
			continue
		}
		if c.Name.Space != wordNS || c.Name.Local != "r" {
			return nil, 0, false, nil
		}
		if start < 0 {
			start = c.Start
		}
		end = c.End
		rPr := ""
		for _, v := range c.Children {
			if v.Name.Space != wordNS || !oneOf(v.Name.Local, "rPr", "t", "tab", "br") {
				return nil, 0, false, nil
			}
			switch v.Name.Local {
			case "rPr":
				rPr = string(raw[v.Start:v.End])
			case "t":
				content := ""
				if v.CloseStart > v.OpenEnd {
					content = xmlUnescape(string(raw[v.OpenEnd:v.CloseStart]))
				}
				n := len([]rune(content))
				if n > 0 {
					tokens = append(tokens, spanToken{start: pos, end: pos + n, kind: "text", text: content, rPr: rPr})
					pos += n
				}
			case "tab":
				tokens = append(tokens, spanToken{start: pos, end: pos + 1, kind: "tab", raw: string(raw[v.Start:v.End]), rPr: rPr})
				pos++
			case "br":
				tokens = append(tokens, spanToken{start: pos, end: pos + 1, kind: "br", raw: string(raw[v.Start:v.End]), rPr: rPr})
				pos++
			}
		}
	}
	if start < 0 || len(tokens) == 0 || pos != len([]rune(original)) {
		return nil, 0, false, nil
	}
	sorted := append([]RevisionSpan(nil), spans...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Start < sorted[j].Start })

	var b strings.Builder
	ns := ` xmlns:w="` + wordNS + `"`
	attrs := ` w:author="教创赛专家" w:date="` + date + `"`
	emit := func(tok spanToken, from, to int, deleted bool) {
		if tok.kind == "text" {
			tr := []rune(tok.text)
			frag := string(tr[from-tok.start : to-tok.start])
			if frag == "" {
				return
			}
			tag := "w:t"
			if deleted {
				tag = "w:delText"
			}
			b.WriteString("<w:r>" + tok.rPr + "<" + tag + ` xml:space="preserve">` + xmlEsc(frag) + "</" + tag + "></w:r>")
			return
		}
		b.WriteString("<w:r>" + tok.rPr + tok.raw + "</w:r>")
	}
	cursor, ti, used := 0, 0, 0
	for _, s := range sorted {
		for cursor < s.Start {
			tok := tokens[ti]
			to := min(tok.end, s.Start)
			emit(tok, cursor, to, false)
			cursor = to
			if cursor == tok.end {
				ti++
			}
		}
		if s.End > s.Start {
			b.WriteString(`<w:del` + ns + fmt.Sprintf(` w:id="%d"`, id+used) + attrs + `>`)
			used++
			for cursor < s.End {
				tok := tokens[ti]
				to := min(tok.end, s.End)
				emit(tok, cursor, to, true)
				cursor = to
				if cursor == tok.end {
					ti++
				}
			}
			b.WriteString(`</w:del>`)
		}
		if s.Insert != "" {
			rPr := ""
			if ti < len(tokens) {
				rPr = tokens[ti].rPr
			} else if len(tokens) > 0 {
				rPr = tokens[len(tokens)-1].rPr
			}
			b.WriteString(`<w:ins` + ns + fmt.Sprintf(` w:id="%d"`, id+used) + attrs + `>` + textRun(s.Insert, rPr) + `</w:ins>`)
			used++
		}
	}
	for cursor < pos {
		tok := tokens[ti]
		emit(tok, cursor, tok.end, false)
		cursor = tok.end
		ti++
	}
	out := string(raw[p.Start:start]) + b.String() + string(raw[end:p.End])
	return []byte(out), used, true, nil
}
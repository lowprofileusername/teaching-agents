package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type DocumentParagraph struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Level int    `json:"level"`
	TOC   bool   `json:"toc"`
}

var cnTop = regexp.MustCompile(`^[一二三四五六七八九十百]+[、．.]`)
var cnSub = regexp.MustCompile(`^[（(][一二三四五六七八九十百]+[）)]`)
var numHeading = regexp.MustCompile(`^\d{1,2}[.．、]\s*\S`)
var tocLeader = regexp.MustCompile(`(?:\t|\.{2,}|…{2,}|·{2,}|\s{2,})\s*\d{1,4}\s*$`)
var headingStyle = regexp.MustCompile(`(?i)(?:heading|标题)\s*([1-9])`)

func paragraphHeading(text, style string, outline int) int {
	if strings.Contains(strings.ToLower(style), "toc") {
		return 0
	}
	if outline >= 0 && outline <= 8 {
		return outline + 1
	}
	if m := headingStyle.FindStringSubmatch(style); len(m) > 1 {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	t := strings.TrimSpace(text)
	if len([]rune(t)) > 100 {
		return 0
	}
	if t == "摘要" || t == "参考文献" || cnTop.MatchString(t) {
		return 1
	}
	if cnSub.MatchString(t) {
		return 2
	}
	if numHeading.MatchString(t) {
		return 3
	}
	return 0
}
func docxParagraphs(data []byte) ([]DocumentParagraph, error) {
	z, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil {
		return nil, e
	}
	var document []byte
	styles := map[string]string{}
	for _, f := range z.File {
		if f.Name != "word/document.xml" && f.Name != "word/styles.xml" {
			continue
		}
		if f.UncompressedSize64 > 8<<20 {
			return nil, fmt.Errorf("文档部件超过限制")
		}
		r, e := f.Open()
		if e != nil {
			return nil, e
		}
		b, e := io.ReadAll(io.LimitReader(r, (8<<20)+1))
		r.Close()
		if e != nil || len(b) > 8<<20 {
			return nil, fmt.Errorf("文档部件无法安全读取")
		}
		if f.Name == "word/document.xml" {
			document = b
			continue
		}
		d := xml.NewDecoder(bytes.NewReader(b))
		id := ""
		for {
			t, e := d.Token()
			if e == io.EOF {
				break
			}
			if e != nil {
				return nil, e
			}
			switch x := t.(type) {
			case xml.StartElement:
				if x.Name.Local == "style" {
					for _, a := range x.Attr {
						if a.Name.Local == "styleId" {
							id = a.Value
						}
					}
				}
				if x.Name.Local == "name" && id != "" {
					for _, a := range x.Attr {
						if a.Name.Local == "val" {
							styles[id] = a.Value
						}
					}
				}
			case xml.EndElement:
				if x.Name.Local == "style" {
					id = ""
				}
			}
		}
	}
	if document == nil {
		return nil, fmt.Errorf("缺少正文")
	}
	type frame struct {
		index   int
		text    strings.Builder
		style   string
		outline int
	}
	stack := []*frame{}
	out := []DocumentParagraph{}
	d := xml.NewDecoder(bytes.NewReader(document))
	inText, deleted := false, 0
	index := 0
	for {
		t, e := d.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, e
		}
		switch x := t.(type) {
		case xml.StartElement:
			if x.Name.Local == "p" {
				index++
				stack = append(stack, &frame{index: index, outline: -1})
			}
			if x.Name.Local == "del" {
				deleted++
			}
			if len(stack) == 0 {
				continue
			}
			p := stack[len(stack)-1]
			switch x.Name.Local {
			case "t":
				inText = deleted == 0
			case "tab":
				if deleted == 0 {
					p.text.WriteByte('\t')
				}
			case "br":
				if deleted == 0 {
					p.text.WriteByte('\n')
				}
			case "pStyle":
				for _, a := range x.Attr {
					if a.Name.Local == "val" {
						p.style = a.Value
						if styles[a.Value] != "" {
							p.style = styles[a.Value]
						}
					}
				}
			case "outlineLvl":
				for _, a := range x.Attr {
					if a.Name.Local == "val" {
						p.outline, _ = strconv.Atoi(a.Value)
					}
				}
			}
		case xml.CharData:
			if inText && len(stack) > 0 {
				stack[len(stack)-1].text.Write(x)
			}
		case xml.EndElement:
			if x.Name.Local == "t" {
				inText = false
			}
			if x.Name.Local == "del" {
				deleted--
			}
			if x.Name.Local == "p" && len(stack) > 0 {
				p := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				text := p.text.String()
				toc := strings.Contains(strings.ToLower(p.style), "toc") || tocLeader.MatchString(text)
				out = append(out, DocumentParagraph{fmt.Sprintf("p%d", p.index), text, paragraphHeading(text, p.style, p.outline), toc})
			}
		}
	}
	return out, nil
}

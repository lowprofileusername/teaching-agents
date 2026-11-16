package main

import (
	"strconv"
	"strings"
	"testing"
)

func spanFixture(t *testing.T, body string) []byte {
	t.Helper()
	return zipFixture(t, map[string]string{
		"[Content_Types].xml": `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="xml" ContentType="application/xml"/></Types>`,
		"word/document.xml":   `<w:document xmlns:w="` + wordNS + `"><w:body>` + body + `</w:body></w:document>`,
	})
}

func TestSpanRevisionMidParagraphCrossRuns(t *testing.T) {
	source := reviewFixture(t)
	edits := []ParagraphEdit{{ID: "p1", Original: "教学问题", Comment: "措辞修订", Spans: []RevisionSpan{{Start: 2, End: 4, Delete: "问题", Insert: "难题", Reason: "措辞"}}}}
	out, result, e := buildReviewedDOCX(source, edits, "revisions")
	if e != nil {
		t.Fatal(e)
	}
	if result.Revisions != 1 || result.Spans != 1 || result.Comments != 1 || len(result.Warnings) != 0 {
		t.Fatal(result)
	}
	ps, e := docxParagraphs(out)
	if e != nil || ps[0].Text != "教学难题" {
		t.Fatal(ps, e)
	}
	parts, _ := officeParts(out)
	doc := string(parts["word/document.xml"])
	for _, s := range []string{"<w:delText", "难题", `<w:r><w:rPr><w:b/></w:rPr><w:t xml:space="preserve">教学</w:t></w:r>`} {
		if !strings.Contains(doc, s) {
			t.Fatal("missing", s)
		}
	}
	if strings.Contains(doc, "<w:delText xml:space=\"preserve\">教学问题</w:delText>") {
		t.Fatal("whole paragraph deleted instead of span")
	}
	if !strings.Contains(string(parts["word/customComments.xml"]), "修订明细") {
		t.Fatal("span detail missing in comment")
	}
}

func TestSpanRevisionMultipleInsertDelete(t *testing.T) {
	source := spanFixture(t, `<w:p><w:r><w:t>甲乙丙丁</w:t></w:r></w:p>`)
	edits := []ParagraphEdit{{ID: "p1", Original: "甲乙丙丁", Comment: "多处修订", Spans: []RevisionSpan{
		{Start: 0, End: 1, Delete: "甲", Insert: "戊"},
		{Start: 2, End: 3, Delete: "丙", Insert: ""},
		{Start: 4, End: 4, Delete: "", Insert: "（待补）"},
	}}}
	out, result, e := buildReviewedDOCX(source, edits, "revisions")
	if e != nil {
		t.Fatal(e)
	}
	if result.Spans != 3 {
		t.Fatal(result)
	}
	ps, _ := docxParagraphs(out)
	if ps[0].Text != "戊乙丁（待补）" {
		t.Fatal(ps[0].Text)
	}
}

func TestSpanRevisionCrossRunBoundary(t *testing.T) {
	source := reviewFixture(t)
	edits := []ParagraphEdit{{ID: "p1", Original: "教学问题", Comment: "跨run", Spans: []RevisionSpan{{Start: 1, End: 3, Delete: "学问", Insert: "X"}}}}
	out, _, e := buildReviewedDOCX(source, edits, "revisions")
	if e != nil {
		t.Fatal(e)
	}
	ps, _ := docxParagraphs(out)
	if ps[0].Text != "教X题" {
		t.Fatal(ps[0].Text)
	}
}

func TestSpanRevisionTabAndBreak(t *testing.T) {
	source := spanFixture(t, `<w:p><w:r><w:t>甲</w:t><w:tab/><w:t>乙</w:t><w:br/><w:t>丙</w:t></w:r></w:p>`)
	edits := []ParagraphEdit{{ID: "p1", Original: "甲\t乙\n丙", Comment: "删除制表与换行", Spans: []RevisionSpan{
		{Start: 1, End: 2, Delete: "\t", Insert: ""},
		{Start: 3, End: 4, Delete: "\n", Insert: ""},
	}}}
	out, _, e := buildReviewedDOCX(source, edits, "revisions")
	if e != nil {
		t.Fatal(e)
	}
	ps, _ := docxParagraphs(out)
	if ps[0].Text != "甲乙丙" {
		t.Fatal(ps[0].Text)
	}
	parts, _ := officeParts(out)
	if _, e := xmlSpans(parts["word/document.xml"]); e != nil {
		t.Fatal(e)
	}
}

func TestSpanRevisionComplexFallsBackToComment(t *testing.T) {
	source := reviewFixture(t)
	edits := []ParagraphEdit{{ID: "p2", Original: "图示", Comment: "复杂段落", Spans: []RevisionSpan{{Start: 0, End: 2, Delete: "图示", Insert: "新图示"}}}}
	out, result, e := buildReviewedDOCX(source, edits, "revisions")
	if e != nil {
		t.Fatal(e)
	}
	if result.Revisions != 0 || len(result.Warnings) != 1 {
		t.Fatal(result)
	}
	ps, _ := docxParagraphs(out)
	if ps[1].Text != "图示" {
		t.Fatal("complex paragraph text changed")
	}
	edits[0].Replacement = "新图示"
	out2, result2, e := buildReviewedDOCX(source, edits, "revisions")
	if e != nil || result2.Revisions != 0 || len(result2.Warnings) != 1 {
		t.Fatal(result2, e)
	}
	ps2, _ := docxParagraphs(out2)
	if ps2[1].Text != "图示" {
		t.Fatal("drawing paragraph rewritten")
	}
}

func TestSpanRevisionValidation(t *testing.T) {
	source := reviewFixture(t)
	for _, spans := range [][]RevisionSpan{
		{{Start: 2, End: 4, Delete: "不对", Insert: "难题"}},
		{{Start: 0, End: 3, Delete: "教学问", Insert: "A"}, {Start: 2, End: 4, Delete: "问题", Insert: "B"}},
		{{Start: 2, End: 4, Delete: "问题", Insert: "提升32%"}},
		{{Start: -1, End: 2, Delete: "教", Insert: "A"}},
		{{Start: 2, End: 2, Delete: "", Insert: ""}},
	} {
		edits := []ParagraphEdit{{ID: "p1", Original: "教学问题", Comment: "校验", Spans: spans}}
		if _, _, e := buildReviewedDOCX(source, edits, "revisions"); e == nil {
			t.Fatalf("invalid spans accepted: %+v", spans)
		}
	}
}

func TestSpanRevisionCommentsModeKeepsText(t *testing.T) {
	source := reviewFixture(t)
	edits := []ParagraphEdit{{ID: "p1", Original: "教学问题", Comment: "批注", Spans: []RevisionSpan{{Start: 2, End: 4, Delete: "问题", Insert: "难题"}}}}
	out, result, e := buildReviewedDOCX(source, edits, "comments")
	if e != nil {
		t.Fatal(e)
	}
	if result.Revisions != 0 || result.Spans != 0 {
		t.Fatal(result)
	}
	ps, _ := docxParagraphs(out)
	if ps[0].Text != "教学问题" {
		t.Fatal("comments mode changed text")
	}
	parts, _ := officeParts(out)
	if !strings.Contains(string(parts["word/customComments.xml"]), "修订明细") {
		t.Fatal("span suggestions missing from comment")
	}
}

func TestSpanRevisionIDsUnique(t *testing.T) {
	source := spanFixture(t, `<w:p><w:r><w:t>甲乙</w:t></w:r></w:p><w:p><w:r><w:t>丙丁</w:t></w:r></w:p>`)
	edits := []ParagraphEdit{
		{ID: "p1", Original: "甲乙", Comment: "一", Spans: []RevisionSpan{{Start: 0, End: 1, Delete: "甲", Insert: "戊"}}},
		{ID: "p2", Original: "丙丁", Comment: "二", Spans: []RevisionSpan{{Start: 0, End: 1, Delete: "丙", Insert: "己"}}},
	}
	out, _, e := buildReviewedDOCX(source, edits, "revisions")
	if e != nil {
		t.Fatal(e)
	}
	parts, _ := officeParts(out)
	revIDs := map[string]bool{}
	commentIDs := map[string]bool{}
	root, e := xmlSpans(parts["word/document.xml"])
	if e != nil {
		t.Fatal(e)
	}
	walkXML(root, func(n *XMLSpan) {
		id := attr(n, "id")
		if _, err := strconv.Atoi(id); err != nil || id == "" {
			return
		}
		switch n.Name.Local {
		case "del", "ins":
			if revIDs[id] {
				t.Fatal("duplicate revision w:id", id)
			}
			revIDs[id] = true
		case "commentRangeStart":
			commentIDs[id] = true
		}
	})
	for id := range revIDs {
		if commentIDs[id] {
			t.Fatal("revision id collides with comment id", id)
		}
	}
	croot, e := xmlSpans(parts["word/comments.xml"])
	if e != nil {
		t.Fatal(e)
	}
	seenComment := map[string]bool{}
	walkXML(croot, func(n *XMLSpan) {
		if n.Name.Local == "comment" {
			id := attr(n, "id")
			if seenComment[id] {
				t.Fatal("duplicate comment id", id)
			}
			seenComment[id] = true
			if revIDs[id] {
				t.Fatal("comment id collides with revision id", id)
			}
		}
	})
	ps, _ := docxParagraphs(out)
	if ps[0].Text != "戊乙" || ps[1].Text != "己丁" {
		t.Fatal(ps)
	}
}
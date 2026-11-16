package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func reviewFixture(t *testing.T) []byte {
	var pngData bytes.Buffer
	png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 3, 3)))
	return zipFixture(t, map[string]string{
		"[Content_Types].xml":          `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="xml" ContentType="application/xml"/></Types>`,
		"word/document.xml":            `<w:document xmlns:w="` + wordNS + `" xmlns:a="a" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><w:body><w:p><w:pPr><w:spacing w:after="120"/></w:pPr><w:r><w:rPr><w:b/></w:rPr><w:t>教学</w:t></w:r><w:r><w:t>问题</w:t></w:r></w:p><w:p><w:r><w:drawing><a:blip r:embed="img1"/></w:drawing><w:t>图示</w:t></w:r></w:p><w:p><w:r><w:t>保留段落</w:t></w:r><w:commentRangeStart w:id="8"/><w:commentRangeEnd w:id="8"/><w:r><w:commentReference w:id="8"/></w:r></w:p><w:p/></w:body></w:document>`,
		"word/_rels/document.xml.rels": `<Relationships xmlns="` + relNS + `"><Relationship Id="img1" Type="` + officeRelNS + `image" Target="media/image1.png"/><Relationship Id="rIdC" Type="` + officeRelNS + `comments" Target="customComments.xml"/></Relationships>`,
		"word/customComments.xml":      `<w:comments xmlns:w="` + wordNS + `"><w:comment w:id="8" w:author="原作者"><w:p><w:r><w:t>原有批注</w:t></w:r></w:p></w:comment></w:comments>`, "word/media/image1.png": pngData.String()})
}
func TestWordReviewExportPreservesAndTracks(t *testing.T) {
	source := reviewFixture(t)
	edits := []ParagraphEdit{{ID: "p1", Original: "教学问题", Comment: "明确真实问题", Replacement: "真实教学问题"}, {ID: "p2", Original: "图示", Comment: "补充图例", Replacement: "新的图示"}, {ID: "p4", Original: "", Comment: "空段标注"}}
	out, result, e := buildReviewedDOCX(source, edits, "revisions")
	if e != nil {
		t.Fatal(e)
	}
	if result.Comments != 3 || result.Revisions != 1 || len(result.Warnings) != 1 {
		t.Fatal(result)
	}
	parts, e := officeParts(out)
	if e != nil {
		t.Fatal(e)
	}
	for n, b := range parts {
		if strings.HasSuffix(n, ".xml") || strings.HasSuffix(n, ".rels") {
			if _, e := xmlSpans(b); e != nil {
				t.Fatal(n, e)
			}
		}
	}
	before, _ := officeParts(source)
	if !bytes.Equal(parts["word/media/image1.png"], before["word/media/image1.png"]) {
		t.Fatal("image altered")
	}
	doc := string(parts["word/document.xml"])
	for _, s := range []string{"w:delText", "w:ins", "真实教学问题", "w:b", "w:drawing", "commentRangeStart", "commentReference"} {
		if !strings.Contains(doc, s) {
			t.Fatal("missing", s)
		}
	}
	if !strings.Contains(string(parts["word/customComments.xml"]), "原有批注") || !strings.Contains(string(parts["word/customComments.xml"]), "教创赛专家") {
		t.Fatal("comments lost")
	}
	ps, e := docxParagraphs(out)
	if e != nil || ps[0].Text != "真实教学问题" || ps[1].Text != "图示" {
		t.Fatal(ps, e)
	}
	plain, _, e := buildReviewedDOCX(source, edits, "comments")
	if e != nil {
		t.Fatal(e)
	}
	pp, _ := docxParagraphs(plain)
	if pp[0].Text != "教学问题" {
		t.Fatal("comment mode changed text")
	}
	bad := []ParagraphEdit{{ID: "p1", Original: "其他文件", Comment: "不可串文档"}}
	if _, _, e = buildReviewedDOCX(source, bad, "comments"); e == nil {
		t.Fatal("mismatched source accepted")
	}
}
func TestImageManifestAndVisionPayload(t *testing.T) {
	p, _ := officeParts(reviewFixture(t))
	vs, e := officeVisuals(p, ".docx")
	if e != nil || len(vs) != 1 || vs[0].ParagraphID != "p2" || !vs[0].Supported {
		t.Fatal(vs, e)
	}
	var b bytes.Buffer
	m := image.NewRGBA(image.Rect(0, 0, 4, 4))
	m.Set(1, 1, color.White)
	jpeg.Encode(&b, m, nil)
	url := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
	wire, e := visionMessages([]Message{{"user", "说明"}}, []VisionImage{{"测试图片", url}})
	if e != nil || len(wire) != 2 {
		t.Fatal(wire, e)
	}
	content := wire[1]["content"].([]any)
	if content[1].(map[string]any)["type"] != "image_url" {
		t.Fatal(content)
	}
	if _, e = visionMessages(nil, []VisionImage{{"远程", "http://127.0.0.1/secret"}}); e == nil {
		t.Fatal("external image accepted")
	}
	p["word/media/image1.png"] = []byte("bad")
	vs, _ = officeVisuals(p, ".docx")
	if vs[0].Supported {
		t.Fatal("invalid image silently accepted")
	}
}

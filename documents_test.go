package main

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func zipFixture(t *testing.T, fs map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for n, s := range fs {
		f, e := z.Create(n)
		if e != nil {
			t.Fatal(e)
		}
		_, _ = f.Write([]byte(s))
	}
	_ = z.Close()
	return b.Bytes()
}
func TestPPTXActualOrder(t *testing.T) {
	b := zipFixture(t, map[string]string{"ppt/presentation.xml": `<p:presentation xmlns:p="p" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><p:sldIdLst><p:sldId id="1" r:id="second"/><p:sldId id="2" r:id="first"/></p:sldIdLst></p:presentation>`, "ppt/_rels/presentation.xml.rels": `<Relationships><Relationship Id="first" Target="slides/slide1.xml"/><Relationship Id="second" Target="/ppt/slides/slide2.xml"/></Relationships>`, "ppt/slides/slide1.xml": `<a:p xmlns:a="a"><a:t>后显示</a:t></a:p>`, "ppt/slides/slide2.xml": `<a:p xmlns:a="a"><a:t>先显示</a:t></a:p>`})
	u, e := parseOffice(b, ".pptx")
	if e != nil {
		t.Fatal(e)
	}
	if len(u) != 2 || !strings.Contains(u[0].Text, "先显示") || !strings.Contains(u[1].Text, "后显示") {
		t.Fatal(u)
	}
}
func TestWordParagraphScope(t *testing.T) {
	b := zipFixture(t, map[string]string{"word/document.xml": `<w:document xmlns:w="w"><w:p><w:r><w:t>报告第一段</w:t></w:r></w:p><w:p><w:r><w:t>报告第二段</w:t></w:r></w:p></w:document>`})
	u, e := parseOffice(b, ".docx")
	if e != nil || len(u) != 1 || !strings.Contains(u[0].Label, "非排版页码") || !strings.Contains(u[0].Text, "第二段") {
		t.Fatal(u, e)
	}
}
func TestOfficeInvalid(t *testing.T) {
	if _, e := parseOffice([]byte("invalid"), ".pptx"); e == nil {
		t.Fatal("accepted bad file")
	}
}
func TestThinkingFormats(t *testing.T) {
	for _, f := range []string{"enable_thinking", "thinking", "reasoning_effort", "chat_template_kwargs"} {
		o := ModelOptions{ThinkingMode: "enabled", ThinkingFormat: f, ReasoningEffort: "high", ContextWindow: 32768, MaxTokens: 8192}
		if e := o.normalize(); e != nil {
			t.Fatal(e)
		}
		b := map[string]any{}
		o.apply(b)
		if b[f] == nil || b["max_tokens"] != 8192 {
			t.Fatal(f, b)
		}
	}
	o := ModelOptions{}
	b := map[string]any{}
	o.apply(b)
	if len(b) != 1 {
		t.Fatal("default leaked provider options")
	}
	o = ModelOptions{ContextWindow: 4096, MaxTokens: 8192}
	if o.normalize() == nil {
		t.Fatal("bad budget accepted")
	}
}
func TestExpandedLibraryDocuments(t *testing.T) {
	p := os.Getenv("TEST_LIBRARY_DB")
	if p == "" {
		t.Skip("real library not configured")
	}
	if e := openLibrary(p); e != nil {
		t.Fatal(e)
	}
	db, _, _ := getLibrary()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	hits := knowledgeSearch(ctx, db, "宋晓菲 单视频深度分析 人工智能", "songxiaofei")
	if len(hits) == 0 {
		t.Fatal("no document hits")
	}
	for _, h := range hits {
		if h["document_id"] == nil || h["pdf_page_no"] == nil || h["authority_status"] == nil {
			t.Fatal("missing provenance")
		}
	}
	t.Log("document retrieval count", len(hits))
}

func TestPortableReportPath(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "report", "Codex_成果", "test.pdf")
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	source := "/Volumes/Share/" + projectFolder + "/report/Codex_成果/test.pdf"
	if got := resolveDocumentPath(source, filepath.Join(root, "expert_videos_v1.sqlite3")); got != target {
		t.Fatal(got)
	}
}

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func makeClaudeFixture(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "claude.db")
	db, err := sql.Open(sqliteDriver, p)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	statements := []string{
		`CREATE TABLE video(tag TEXT PRIMARY KEY,speaker TEXT,school TEXT,course TEXT,talk_title TEXT,duration_s REAL,asr_engine TEXT,asr_quality TEXT,method TEXT,provenance TEXT,session TEXT,track TEXT,audio BLOB)`,
		`CREATE TABLE page(id INTEGER PRIMARY KEY,tag TEXT,page_no INT,title TEXT,ocr_text TEXT,section TEXT,t_start REAL,t_end REAL,time_ranges TEXT,main_message TEXT,elements TEXT,function TEXT,relation TEXT,visual TEXT,evidence TEXT,borrow TEXT,migrate TEXT,speech_text TEXT,speech_sentences TEXT,image BLOB,audio BLOB)`,
		`CREATE TABLE utterance(id INTEGER PRIMARY KEY,tag TEXT,page_no INT,t_start REAL,t_end REAL,text TEXT)`,
		`CREATE TABLE insight(id INTEGER PRIMARY KEY,kind TEXT,title TEXT,body TEXT,sources TEXT)`,
		`CREATE TABLE errata(tag TEXT,wrong TEXT,right TEXT,found_by TEXT)`,
		`CREATE TABLE readme(key TEXT,body TEXT)`,
		`CREATE TABLE flash(id INTEGER PRIMARY KEY,tag TEXT,mid REAL,text TEXT,image BLOB)`,
		`CREATE TABLE second_full(tag TEXT,t INT,ocr_text TEXT)`,
		`CREATE TABLE second(tag TEXT,t INT,ocr_text TEXT)`,
		`INSERT INTO video VALUES('song','宋晓菲','示例学校','示例课程','案例',100,'ASR','未听校','','','','',X'4F676753')`,
		`INSERT INTO video SELECT 'li','李静',school,course,talk_title,duration_s,asr_engine,asr_quality,method,provenance,session,track,audio FROM video`,
		`INSERT INTO page(id,tag,page_no,title,ocr_text,t_start,t_end,time_ranges,main_message,borrow,image) VALUES(1,'song',12,'人工智能','页面独有文字 人工智能',10,50,'[[10,20],[40,50]]','这是研究分析','这是迁移建议',X'FFD8FF')`,
		`INSERT INTO page(id,tag,page_no,title,ocr_text,t_start,t_end,time_ranges) VALUES(2,'li',12,'人工智能','页面文字 人工智能',10,20,'[[10,20]]')`,
		`INSERT INTO utterance VALUES(1,'song',12,11.5,13.5,'口述独有内容')`,
		`INSERT INTO utterance VALUES(2,'song',12,41,44,'回看页口述')`,
		`INSERT INTO insight VALUES(1,'研究','共同打法','研究者的归纳','宋晓菲')`,
		`INSERT INTO errata VALUES('song','错词','正词','来源库')`,
		`INSERT INTO readme VALUES('usage','Synthetic fixture only')`,
		`INSERT INTO flash VALUES(1,'song',4.2,'闪现字',X'FFD8FF')`,
		`INSERT INTO second VALUES('song',5,'旧闪现字')`,
		`INSERT INTO second_full VALUES('song',5,'新闪现字')`,
	}
	for _, s := range statements {
		if _, err = db.Exec(s); err != nil {
			t.Fatal(err, s)
		}
	}
	return p
}

func TestClaudeLibraryAdapter(t *testing.T) {
	p := makeClaudeFixture(t)
	before, _ := os.ReadFile(p)
	if err := openLibrary(p); err != nil {
		t.Fatal(err)
	}
	db, _, _ := getLibrary()
	ctx := context.Background()
	if !isClaudeLibrary(db) {
		t.Fatal("not detected")
	}
	if _, err := db.Exec(`UPDATE main.video SET speaker='wrong'`); err == nil {
		t.Fatal("source is writable")
	}
	page, err := enrichPage(ctx, db, "song:P12")
	if err != nil {
		t.Fatal(err)
	}
	if page["page_text"] != "页面独有文字 人工智能" || page["original_page_no"] != nil {
		t.Fatal("mixed evidence or invented page number")
	}
	if len(page["occurrences"].([]map[string]any)) != 2 || len(page["utterances"].([]map[string]any)) != 2 {
		t.Fatal("lost revisit or transcript")
	}
	mime, b, err := claudeAsset(ctx, db, "claude:page-image:1")
	if err != nil || mime != "image/jpeg" || len(b) != 3 {
		t.Fatal(mime, err)
	}
	a := &App{}
	w := httptest.NewRecorder()
	a.librarySearch(w, httptest.NewRequest("POST", "/", strings.NewReader(`{"q":"人工智能赛道"}`)))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "song:P12") || !strings.Contains(w.Body.String(), "li:P12") {
		t.Fatal(w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	a.librarySearch(w, httptest.NewRequest("POST", "/", strings.NewReader(`{"q":"口述独有内容原话"}`)))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "song:P12") {
		t.Fatal(w.Body.String())
	}
	w = httptest.NewRecorder()
	claudeAudio(w, httptest.NewRequest("POST", "/", nil), db, "song", 42)
	if !strings.Contains(w.Body.String(), `"t0":0`) {
		t.Fatal("revisit audio time axis changed")
	}
	raw := claudeRawOCR(ctx, db, "闪现字", "song")
	b, _ = json.Marshal(raw)
	if !strings.Contains(string(b), "新闪现字") || strings.Contains(string(b), "旧闪现字") {
		t.Fatal(string(b))
	}
	db.Close()
	after, _ := os.ReadFile(p)
	if string(before) != string(after) {
		t.Fatal("source bytes changed")
	}
}

func TestRealClaudeLibrary(t *testing.T) {
	path := os.Getenv("TEST_CLAUDE_LIBRARY_DB")
	if path == "" {
		t.Skip("private source not configured")
	}
	before, _ := os.Stat(path)
	start := time.Now()
	if err := openLibrary(path); err != nil {
		t.Fatal(err)
	}
	t.Log("connect", time.Since(start))
	db, _, _ := getLibrary()
	ctx := context.Background()
	var n int
	db.QueryRow("SELECT count(*) FROM video").Scan(&n)
	if n != 28 {
		t.Fatal(n)
	}
	all, err := rowsJSON(ctx, db, `SELECT id FROM page`)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range all {
		p, err := enrichPage(ctx, db, r["id"].(string))
		if err != nil {
			t.Fatal(r, err)
		}
		if p["teacher"] == nil || len(p["occurrences"].([]map[string]any)) == 0 {
			t.Fatal(r)
		}
	}
	a := &App{}
	for _, query := range []string{"宋晓菲和李静的人工智能教学有什么区别？", "人工智能赛道的创新报告模板", "九位一等奖的共同打法", "新文科赛道怎么写报告", "金海京第12页", "快闪OCR"} {
		start = time.Now()
		body, _ := json.Marshal(map[string]any{"q": query})
		w := httptest.NewRecorder()
		a.librarySearch(w, httptest.NewRequest("POST", "/", strings.NewReader(string(body))))
		if w.Code != 200 {
			t.Fatal(query, w.Body.String())
		}
		var result struct {
			Pages []map[string]any `json:"pages"`
		}
		json.Unmarshal(w.Body.Bytes(), &result)
		if len(result.Pages) == 0 {
			t.Fatal(query, "no pages")
		}
		if strings.Contains(query, "宋晓菲和李静") {
			tags := map[string]bool{}
			for _, p := range result.Pages {
				tags[p["tag"].(string)] = true
			}
			if !tags["song"] || !tags["li"] {
				t.Fatal(tags)
			}
		}
		t.Logf("%s %v %d pages", query, time.Since(start), len(result.Pages))
	}
	p, err := enrichPage(ctx, db, "song:P12")
	if err != nil {
		t.Fatal(err)
	}
	imgs := p["images"].([]map[string]any)
	if len(imgs) == 0 {
		t.Fatal("no image")
	}
	_, b, err := claudeAsset(ctx, db, imgs[0]["asset_id"].(string))
	if err != nil || len(b) < 3 || b[0] != 255 || b[1] != 216 {
		t.Fatal("image", err)
	}
	_, b, err = claudeAsset(ctx, db, "claude:video-audio:song")
	if err != nil || !strings.HasPrefix(string(b[:min(4, len(b))]), "OggS") {
		t.Fatal("audio", err)
	}
	after, _ := os.Stat(path)
	if before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("source changed")
	}
	t.Logf("all %d pages enriched, screenshot/audio retrieved, source unchanged", len(all))
}

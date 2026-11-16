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
)

func TestLectureTimelinePreservesRevisitsAndDoesNotLoadMedia(t *testing.T) {
	db, e := sql.Open(sqliteDriver, filepath.Join(t.TempDir(), "lecture.sqlite3"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	schema, e := os.ReadFile("tests/fixture_schema.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(string(schema)); e != nil {
		t.Fatal(e)
	}
	for _, q := range []string{`INSERT INTO video(tag,title,teacher,duration)VALUES('v','讲座','教师',20)`, `INSERT INTO page(id,tag,page_no,title)VALUES('v:R1','v',1,'页一')`, `INSERT INTO page_occurrence VALUES('a','v:R1',0,5),('b','v:R1',10,15)`, `INSERT INTO page_image VALUES('v:R1','image1',2,'','代表截图')`, `INSERT INTO utterance(id,tag,t0,t1,text,version)VALUES(1,'v',1,3,'测试原话','test-asr')`} {
		if _, e = db.Exec(q); e != nil {
			t.Fatal(e)
		}
	}
	x, e := lectureTimeline(context.Background(), db, "v")
	if e != nil {
		t.Fatal(e)
	}
	if len(x["entries"].([]map[string]any)) != 2 {
		t.Fatal("lost revisits")
	}
	b, _ := json.Marshal(x)
	if strings.Contains(string(b), `"content"`) {
		t.Fatal("timeline loaded media blob")
	}
	if len(x["images"].([]map[string]any)) != 1 {
		t.Fatal("missing image index")
	}
}
func TestRealLectureMediaIndexes(t *testing.T) {
	if os.Getenv("TEST_REPLAY_DATABASES") == "" {
		t.Skip("optional real read-only integration")
	}
	for _, path := range strings.Split(os.Getenv("TEST_REPLAY_DATABASES"), "|") {
		before, e := os.Stat(path)
		if e != nil {
			t.Fatal(e)
		}
		if e = openLibrary(path); e != nil {
			t.Fatal(e)
		}
		db, _, _ := getLibrary()
		tag := "songxiaofei"
		if isClaudeLibrary(db) {
			tag = "song"
		}
		x, e := lectureTimeline(context.Background(), db, tag)
		if e != nil {
			t.Fatal(e)
		}
		entries := len(x["entries"].([]map[string]any))
		if entries == 0 || len(x["images"].([]map[string]any)) == 0 || len(x["utterances"].([]map[string]any)) == 0 {
			t.Fatal("missing replay data")
		}
		w := httptest.NewRecorder()
		(&App{}).libraryAudio(w, httptest.NewRequest("POST", "/api/audio", strings.NewReader(`{"tag":"`+tag+`","time":121}`)))
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		after, _ := os.Stat(path)
		if before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
			t.Fatal("changed source")
		}
		t.Log(path, entries, "intervals and audio lookup passed")
	}
}

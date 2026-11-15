package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"testing"
)

func TestLosslessMediaAndCorruption(t *testing.T) {
	db, e := sql.Open(sqliteDriver, ":memory:")
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.Exec("CREATE TABLE media_encoding(table_name TEXT,record_id TEXT,codec TEXT,raw_sha256 TEXT,raw_bytes INT)")
	raw := bytes.Repeat([]byte("原始PDF/音频数据"), 100)
	sum := sha256.Sum256(raw)
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(raw)
	w.Close()
	db.Exec("INSERT INTO media_encoding VALUES(?,?,?,?,?)", "asset", "sample", "zlib", hex.EncodeToString(sum[:]), len(raw))
	out, e := decodeMedia(context.Background(), db, "asset", "sample", b.Bytes())
	if e != nil || !bytes.Equal(raw, out) {
		t.Fatal(e)
	}
	if _, e = decodeMedia(context.Background(), db, "asset", "sample", []byte("corrupt")); e == nil {
		t.Fatal("corruption accepted")
	}
}

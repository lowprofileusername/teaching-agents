package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestTextCacheIsolationAndInvalidation(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cache.sqlite3")
	a, e := sql.Open(sqliteDriver, p)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	a.SetMaxOpenConns(1)
	a.Exec("CREATE TABLE data(v TEXT)")
	a.Exec("INSERT INTO data VALUES('original')")
	first, e := cachedTextRows(context.Background(), a, "SELECT v FROM data")
	if e != nil {
		t.Fatal(e)
	}
	first[0]["v"] = "mutation"
	second, _ := cachedTextRows(context.Background(), a, "SELECT v FROM data")
	if second[0]["v"] != "original" {
		t.Fatal("cached rows mutated")
	}
	b, _ := sql.Open(sqliteDriver, p)
	defer b.Close()
	if _, e = b.Exec("UPDATE data SET v='updated'"); e != nil {
		t.Fatal(e)
	}
	third, _ := cachedTextRows(context.Background(), a, "SELECT v FROM data")
	if third[0]["v"] != "updated" {
		t.Fatal("stale database cache")
	}
}

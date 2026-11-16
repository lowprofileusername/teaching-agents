package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// Separate mutable user workspace store; never writes to lecture databases.
func (a *App) workspaceDB() (*sql.DB, error) {
	if a.profiles == nil {
		return nil, errors.New("用户配置目录尚未就绪")
	}
	dir := filepath.Dir(a.profiles.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open(sqliteDriver, filepath.Join(dir, "workbench_history.sqlite3"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`PRAGMA busy_timeout=4000; PRAGMA journal_mode=DELETE; PRAGMA synchronous=FULL;
 CREATE TABLE IF NOT EXISTS workspace(id TEXT PRIMARY KEY,title TEXT NOT NULL,revision INTEGER NOT NULL,updated TEXT NOT NULL,state TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS window_layout(name TEXT PRIMARY KEY,geometry TEXT NOT NULL);`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

var workspaceID = regexp.MustCompile(`^[a-zA-Z0-9_-]{8,80}$`)

func (a *App) workspaceAPI(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Action   string          `json:"action"`
		ID       string          `json:"id"`
		Title    string          `json:"title"`
		Revision int             `json:"revision"`
		State    json.RawMessage `json:"state"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<20)).Decode(&x) != nil {
		fail(w, 400, "工作记录无效或超过32MB；未覆盖已有记录")
		return
	}
	db, e := a.workspaceDB()
	if e != nil {
		fail(w, 500, "无法保存工作记录："+e.Error())
		return
	}
	defer db.Close()
	if x.Action == "list" {
		rows, e := db.Query(`SELECT id,title,revision,updated FROM workspace ORDER BY updated DESC LIMIT 100`)
		if e != nil {
			fail(w, 500, e.Error())
			return
		}
		defer rows.Close()
		list := []map[string]any{}
		for rows.Next() {
			var id, title, at string
			var rev int
			if e = rows.Scan(&id, &title, &rev, &at); e != nil {
				break
			}
			list = append(list, map[string]any{"id": id, "title": title, "revision": rev, "updated": at})
		}
		if e != nil || rows.Err() != nil {
			fail(w, 500, "读取工作记录失败")
			return
		}
		jsonOut(w, 200, map[string]any{"sessions": list, "directory": filepath.Dir(a.profiles.path)})
		return
	}
	if !workspaceID.MatchString(x.ID) {
		fail(w, 400, "工作记录编号无效")
		return
	}
	switch x.Action {
	case "load":
		var raw, title, at string
		var rev int
		e = db.QueryRow(`SELECT state,title,revision,updated FROM workspace WHERE id=?`, x.ID).Scan(&raw, &title, &rev, &at)
		if e == sql.ErrNoRows {
			fail(w, 404, "工作记录不存在")
			return
		}
		if e != nil {
			fail(w, 500, e.Error())
			return
		}
		jsonOut(w, 200, map[string]any{"id": x.ID, "title": title, "revision": rev, "updated": at, "state": json.RawMessage(raw)})
	case "save":
		var meta struct {
			Version int `json:"version"`
		}
		if json.Unmarshal(x.State, &meta) != nil || meta.Version != 1 {
			fail(w, 400, "不支持的工作记录格式")
			return
		}
		if len([]rune(x.Title)) > 200 {
			fail(w, 400, "名称过长")
			return
		}
		at := time.Now().UTC().Format(time.RFC3339Nano)
		var result sql.Result
		if x.Revision == 0 {
			result, e = db.Exec(`INSERT OR IGNORE INTO workspace VALUES(?,?,1,?,?)`, x.ID, x.Title, at, string(x.State))
		} else {
			result, e = db.Exec(`UPDATE workspace SET title=?,revision=revision+1,updated=?,state=? WHERE id=? AND revision=?`, x.Title, at, string(x.State), x.ID, x.Revision)
		}
		if e != nil {
			fail(w, 500, "工作记录写入失败："+e.Error())
			return
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			fail(w, 409, "此记录已在另一窗口更新，未覆盖；请另存副本")
			return
		}
		jsonOut(w, 200, map[string]any{"id": x.ID, "revision": x.Revision + 1, "updated": at})
	default:
		fail(w, 400, "未知工作记录操作")
	}
}

type windowGeometry struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

func clampGeometry(g windowGeometry, x, y, width, height int) windowGeometry {
	g.Width = min(max(g.Width, 800), width)
	g.Height = min(max(g.Height, 600), height)
	g.X = min(max(g.X, x), x+width-g.Width)
	g.Y = min(max(g.Y, y), y+height-g.Height)
	return g
}
func (a *App) loadGeometry(name string) (windowGeometry, bool) {
	var g windowGeometry
	db, e := a.workspaceDB()
	if e != nil {
		return g, false
	}
	defer db.Close()
	var raw string
	e = db.QueryRow(`SELECT geometry FROM window_layout WHERE name=?`, name).Scan(&raw)
	if e != nil || json.Unmarshal([]byte(raw), &g) != nil || g.Width < 100 || g.Height < 100 {
		return g, false
	}
	return g, true
}
func (a *App) saveGeometry(name string, g windowGeometry) error {
	db, e := a.workspaceDB()
	if e != nil {
		return e
	}
	defer db.Close()
	b, e := json.Marshal(g)
	if e != nil {
		return e
	}
	_, e = db.Exec(`INSERT INTO window_layout VALUES(?,?) ON CONFLICT(name) DO UPDATE SET geometry=excluded.geometry`, name, string(b))
	if e != nil {
		return fmt.Errorf("保存窗口位置：%w", e)
	}
	return nil
}
